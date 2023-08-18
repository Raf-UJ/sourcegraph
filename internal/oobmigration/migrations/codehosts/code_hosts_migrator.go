package codehosts

import (
	"context"
	"database/sql"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/sourcegraph/log"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/encryption"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/oobmigration"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type codeHostsMigrator struct {
	logger log.Logger
	store  *basestore.Store
	key    encryption.Key
}

var _ oobmigration.Migrator = &codeHostsMigrator{}

func NewMigratorWithDB(store *basestore.Store, key encryption.Key) *codeHostsMigrator {
	return &codeHostsMigrator{
		logger: log.Scoped("codeHostsMigrator", ""),
		store:  store,
		key:    key,
	}
}

func (m *codeHostsMigrator) ID() int                 { return 24 }
func (m *codeHostsMigrator) Interval() time.Duration { return 3 * time.Second }

// Progress returns the percentage (ranged [0, 1]) of external services that were migrated to the code_hosts table.
func (m *codeHostsMigrator) Progress(ctx context.Context, _ bool) (float64, error) {
	progress, _, err := basestore.ScanFirstFloat(m.store.Query(ctx, sqlf.Sprintf(codeHostsMigratorProgressQuery)))
	return progress, err
}

// Note: We explicitly also migrate deleted external services here, so that we can be sure by 5.3
// that there will be no more external_services without an associated code host so we can make
// the code_host_id column non-nullable.
const codeHostsMigratorProgressQuery = `
SELECT
	CASE c2.count WHEN 0 THEN 1 ELSE
		CAST(c1.count AS float) / CAST(c2.count AS float)
	END
FROM
	(SELECT COUNT(*) AS count FROM external_services WHERE code_host_id IS NOT NULL) c1,
	(SELECT COUNT(*) AS count FROM external_services WHERE code_host_id IS NULL) c2
`

// Up loads a set of external services without a populated code_host_id column and
// upserts a code_hosts entry to fill the value for the host.
func (m *codeHostsMigrator) Up(ctx context.Context) (err error) {
	tx, err := m.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Done(err)
	}()

	// First, read the currently configured value for gitMaxCodehostRequestsPerSecond from
	// the site config. This value needs to be transferred to any code host that we create.
	gitMaxCodehostRequestsPerSecond := 0
	{
		row := tx.QueryRow(ctx, sqlf.Sprintf(currentSiteConfigQuery))
		var siteConfigContents string
		if err := row.Scan(&siteConfigContents); err != nil {
			// No site config could exist, skip in this case.
			if err != sql.ErrNoRows {
				return errors.Wrap(err, "failed to read current site config")
			}
		}
		if siteConfigContents != "" {
			var cfg siteConfiguration
			if err := jsonc.Unmarshal(siteConfigContents, &cfg); err != nil {
				return errors.Wrap(err, "failed to parse current site config")
			}
			if cfg.GitMaxCodehostRequestsPerSecond != nil {
				gitMaxCodehostRequestsPerSecond = *cfg.GitMaxCodehostRequestsPerSecond
			}
		}
	}

	type svc struct {
		ID           int
		Kind, Config string
	}
	svcs, err := func() (svcs []svc, err error) {
		// First, we load ALL external_services. This should be << 50 for most instances
		// so this should not cause bigger issues.
		rows, err := tx.Query(ctx, sqlf.Sprintf(listAllExternalServicesQuery))
		if err != nil {
			return nil, err
		}
		defer func() { err = basestore.CloseRows(rows, err) }()

		for rows.Next() {
			var id int
			var kind, config, keyID string
			if err := rows.Scan(&id, &kind, &config, &keyID); err != nil {
				return nil, err
			}
			config, err = encryption.MaybeDecrypt(ctx, m.key, config, keyID)
			if err != nil {
				return nil, err
			}

			svcs = append(svcs, svc{ID: id, Kind: kind, Config: config})
		}

		return svcs, nil
	}()
	if err != nil {
		return err
	}

	// Nothing more to migrate!
	if len(svcs) == 0 {
		return nil
	}

	// Look at the first unmigrated external service.
	current := svcs[0]
	currentHostURL, err := UniqueCodeHostIdentifier(current.Kind, current.Config)
	if err != nil {
		return err
	}
	lowestRateLimitPerHour, isLowestRateLimitDefault, err := ExtractRateLimit(current.Config, current.Kind)
	if err != nil && !errors.HasType(err, ErrRateLimitUnsupported{}) {
		return err
	}
	if lowestRateLimitPerHour < 0 {
		lowestRateLimitPerHour = 0
	}
	svcsWithSameHost := []int{current.ID}

	// Find all other external services for the same (kind, url).
	for _, o := range svcs[1:] {
		if o.Kind != current.Kind {
			continue
		}
		haveHostURL, err := UniqueCodeHostIdentifier(o.Kind, o.Config)
		if err != nil {
			return err
		}

		if haveHostURL != currentHostURL {
			// TODO: Test for missing trailing slash.
			continue
		}

		svcsWithSameHost = append(svcsWithSameHost, o.ID)
		// Find the smallest configured rate limit for the given host.
		rateLimit, isDefaultRateLimit, err := ExtractRateLimit(current.Config, current.Kind)
		if err != nil && !errors.HasType(err, ErrRateLimitUnsupported{}) {
			return err
		}
		if isDefaultRateLimit {
			continue
		}
		if rateLimit >= 0 && rateLimit < lowestRateLimitPerHour {
			lowestRateLimitPerHour = rateLimit
			isLowestRateLimitDefault = false
		}
	}

	var apiInterval int
	var apiRateLimit int
	if lowestRateLimitPerHour != rate.Inf && lowestRateLimitPerHour != 0. && !isLowestRateLimitDefault {
		apiInterval = 60 * 60 // limits used to always be one hour.
		apiRateLimit = int(lowestRateLimitPerHour * 60 * 60)
	}

	var gitInterval int
	if gitMaxCodehostRequestsPerSecond > 0 {
		gitInterval = 1 // always per second in site-config.
	}

	row := tx.QueryRow(ctx, sqlf.Sprintf(
		upsertCodeHostQuery,
		current.Kind,
		currentHostURL,
		NewNullInt(apiRateLimit),
		NewNullInt(apiInterval),
		NewNullInt(gitMaxCodehostRequestsPerSecond),
		NewNullInt(gitInterval),
		currentHostURL,
	))

	var codeHostID int
	if err := row.Scan(&codeHostID); err != nil {
		return errors.Wrap(err, "failed to upsert code host")
	}

	return tx.Exec(ctx, sqlf.Sprintf(setCodeHostIDOnExternalServiceQuery, codeHostID, pq.Array(svcsWithSameHost)))
}

const listAllExternalServicesQuery = `
SELECT
	id, kind, config, encryption_key_id
FROM
	external_services
WHERE
	code_host_id IS NULL
ORDER BY id
FOR UPDATE
`

const currentSiteConfigQuery = `
SELECT contents FROM critical_and_site_config WHERE type = 'site' ORDER BY id DESC LIMIT 1
`

const upsertCodeHostQuery = `
WITH inserted AS (
	INSERT INTO
		code_hosts
			(kind, url, api_rate_limit_quota, api_rate_limit_interval_seconds, git_rate_limit_quota, git_rate_limit_interval_seconds)
	VALUES
			(%s, %s, %s, %s, %s, %s)
	ON CONFLICT (url) DO NOTHING
	RETURNING
		id
)
SELECT
	id
FROM inserted

UNION

SELECT
	id
FROM code_hosts
WHERE url = %s
`

const setCodeHostIDOnExternalServiceQuery = `
UPDATE external_services
SET
	code_host_id = %s
WHERE
	id = ANY(%s)
`

func (*codeHostsMigrator) Down(context.Context) error {
	// non-destructive
	return nil
}

type siteConfiguration struct {
	GitMaxCodehostRequestsPerSecond *int `json:"gitMaxCodehostRequestsPerSecond,omitempty"`
}
