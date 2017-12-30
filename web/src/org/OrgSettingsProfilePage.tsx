import LoaderIcon from '@sourcegraph/icons/lib/Loader'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { of } from 'rxjs/observable/of'
import { concat } from 'rxjs/operators/concat'
import { delay } from 'rxjs/operators/delay'
import { distinctUntilChanged } from 'rxjs/operators/distinctUntilChanged'
import { mergeMap } from 'rxjs/operators/mergeMap'
import { startWith } from 'rxjs/operators/startWith'
import { switchMap } from 'rxjs/operators/switchMap'
import { tap } from 'rxjs/operators/tap'
import { Subject } from 'rxjs/Subject'
import { Subscription } from 'rxjs/Subscription'
import { PageTitle } from '../components/PageTitle'
import { eventLogger } from '../tracking/eventLogger'
import { updateOrg } from './backend'

interface Props extends RouteComponentProps<any> {
    org: GQL.IOrg
    user: GQL.IUser
}

interface State {
    slackWebhookURL: string
    displayName: string
    loading: boolean
    updated: boolean
    error?: string
}

/**
 * The organizations settings page
 */
export class OrgSettingsProfilePage extends React.PureComponent<Props, State> {
    private orgChanges = new Subject<GQL.IOrg>()
    private submits = new Subject<void>()
    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)

        this.state = {
            slackWebhookURL: props.org.slackWebhookURL || '',
            displayName: props.org.displayName || '',
            loading: false,
            updated: false,
        }
    }

    public componentDidMount(): void {
        this.subscriptions.add(
            this.orgChanges
                .pipe(
                    distinctUntilChanged(),
                    tap(org => eventLogger.logViewEvent('OrgProfile', { organization: { org_name: org.name } }))
                )
                .subscribe()
        )
        this.orgChanges.next(this.props.org)

        this.subscriptions.add(
            this.submits
                .pipe(
                    switchMap(() =>
                        updateOrg(this.props.org.id, this.state.displayName, this.state.slackWebhookURL).pipe(
                            mergeMap(() =>
                                // Reset email, reenable submit button, flash "updated" text
                                of<Partial<State>>({ loading: false, updated: true })
                                    // Hide "updated" text again after 1s
                                    .pipe(concat(of<Partial<State>>({ updated: false }).pipe(delay(1000))))
                            ),
                            // Disable button while loading
                            startWith<Partial<State>>({ loading: true })
                        )
                    )
                )
                .subscribe((state: State) => this.setState(state))
        )
        // TODO(sqs): handle errors
    }

    public componentWillReceiveProps(props: Props): void {
        if (props.org !== this.props.org) {
            this.orgChanges.next(props.org)
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="org-settings-profile-page">
                <PageTitle title={this.props.org.name} />
                <div className="org-settings-profile-page__header">
                    <h2>{this.props.org.name}</h2>
                </div>
                <form className="org-settings-profile-page" onSubmit={this.onSubmit}>
                    <div className="form-group">
                        <label>Display name</label>
                        <input
                            type="text"
                            className="form-control org-settings-profile-page__display-name"
                            placeholder="Organization name"
                            onChange={this.onDisplayNameFieldChange}
                            value={this.state.displayName}
                            spellCheck={false}
                            size={60}
                        />
                    </div>
                    <div className="form-group">
                        <label>Slack webhook URL</label>
                        <input
                            type="url"
                            className="form-control org-settings-profile-page__slack-webhook-url"
                            placeholder=""
                            onChange={this.onSlackWebhookURLFieldChange}
                            value={this.state.slackWebhookURL}
                            spellCheck={false}
                            size={60}
                        />
                        <small className="form-text">
                            Integrate Sourcegraph's code comments and org updates with your team's Slack channel! Go to
                            &lt;your-workspace-url&gt;.slack.com/apps/new/A0F7XDUAZ-incoming-webhooks
                        </small>
                    </div>
                    <button
                        type="submit"
                        disabled={this.state.loading}
                        className="btn btn-primary org-settings-profile-page__submit-button"
                    >
                        Update
                    </button>
                    {this.state.loading && <LoaderIcon className="icon-inline" />}
                    <div
                        className={
                            'org-settings-profile-page__updated-text' +
                            (this.state.updated ? ' org-settings-profile-page__updated-text--visible' : '')
                        }
                    >
                        <small>Updated!</small>
                    </div>
                    {this.state.error && <div className="alert alert-danger">{this.state.error}</div>}
                </form>
            </div>
        )
    }

    private onSlackWebhookURLFieldChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        this.setState({ slackWebhookURL: e.target.value })
    }

    private onDisplayNameFieldChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        this.setState({ displayName: e.target.value })
    }

    private onSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault()
        this.submits.next()
    }
}
