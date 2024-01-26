import React from 'react'

import { mdiClose } from '@mdi/js'
import classNames from 'classnames'

import { useExperimentalFeatures } from '@sourcegraph/shared/src/settings/settings'
import { useTemporarySetting } from '@sourcegraph/shared/src/settings/temporary'
import { Code, H2, Icon, Link, Text } from '@sourcegraph/wildcard'

import { MarketingBlock } from '../../../components/MarketingBlock'
import { NewStarsIcon } from '../../../search/results/components/search-results-info-bar/NewStarsIcon'

import styles from './KeywordSearchCtaSection.module.scss'

interface KeywordSearchCtaSection {
    className?: string
}

export const KeywordSearchCtaSection: React.FC<KeywordSearchCtaSection> = ({ className }) => {
    const keywordSearchEnabled = useExperimentalFeatures(features => features.keywordSearch)
    const [isDismissed = true, setIsDismissed] = useTemporarySetting('search.homepage.keywordCta.dismissed', false)
    if (!keywordSearchEnabled || isDismissed) {
        return null
    }

    return (
        <MarketingBlock contentClassName={classNames('flex-grow-1 d-flex justify-content-between p-4', styles.card)}>
            <div>
                <H2>
                    <NewStarsIcon aria-hidden={true} /> New, improved search
                </H2>
                <Text>
                    <ul>
                        <li>
                            The search bar now supports <b>keyword search</b>, where terms match broadly across the file
                            contents and path
                        </li>
                        <li>The new behavior ANDs terms together instead of searching literally by default </li>
                        <li>
                            To search literally, wrap the query in quotes like <Code>"Error 101: service failed"</Code>
                        </li>
                    </ul>
                    <Link
                        to="https://sourcegraph.com/docs/code-search/queries#keyword-search-default"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        Read the docs
                    </Link>{' '}
                    to learn more.
                </Text>
            </div>
            <Icon
                svgPath={mdiClose}
                aria-label="Close keyword search explanation"
                className={classNames(styles.closeButton)}
                onClick={() => setIsDismissed(true)}
            />
        </MarketingBlock>
    )
}
