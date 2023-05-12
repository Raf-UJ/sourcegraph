import { FC, useState, useCallback, useRef } from 'react'

import { useLocation } from 'react-router-dom'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { Alert, Container, Button, H2, Input, Label, Text } from '@sourcegraph/wildcard'

import { PageTitle } from '../PageTitle'

export interface AddGitHubPageProps extends TelemetryProps {}

interface stateResponse {
    state: string
    webhookUUID: string
}

/**
 * Page for choosing a service kind and variant to add, among the available options.
 */
export const AddGitHubAppPage: FC<AddGitHubPageProps> = () => {
    const { search } = useLocation()
    const ref = useRef<HTMLFormElement>(null)
    const formInput = useRef<HTMLInputElement>(null)
    const [name, setName] = useState<string>('Sourcegraph')
    const [url, setUrl] = useState<string>('https://github.com')
    const [org, setOrg] = useState<string>('')
    const [error, setError] = useState<any>(null)

    let alert = null
    const id = new URLSearchParams(search).get('id')
    if (id) {
        // TODO: handle this on a different page
        alert = <Alert variant="info">GitHub App installed successfully.</Alert>
    }

    const baseUrl = window.location.origin
    const getManifest = useCallback(
        (name: string, webhookURL: string): string =>
            JSON.stringify({
                name: name.trim(),
                url: baseUrl,
                hook_attributes: {
                    url: webhookURL,
                },
                redirect_url: new URL('/.auth/githubapp/redirect', baseUrl).href,
                setup_url: new URL('/.auth/githubapp/setup', baseUrl).href,
                callback_urls: [new URL('/.auth/github/callback', baseUrl).href],
                setup_on_update: true,
                public: false,
                default_permissions: {
                    contents: 'read',
                    emails: 'read',
                    members: 'read',
                    metadata: 'read',
                },
                default_events: [
                    'repository',
                    'public',
                    'member',
                    'membership',
                    'organization',
                    'team',
                    'team_add',
                    'meta',
                    'push',
                ],
            }),
        [baseUrl]
    )

    const createActionUrl = useCallback(
        (state: string): string => {
            let prefix = 'settings/apps/new'
            const trimmedOrg = org.trim()
            if (trimmedOrg !== '') {
                prefix = `organizations/${encodeURIComponent(trimmedOrg)}/settings/apps/new`
            }

            const trimmedUrl = url.trim()
            const baseUrl = trimmedUrl.endsWith('/') ? trimmedUrl : `${trimmedUrl}/`
            return new URL(`${prefix}?state=${state}`, baseUrl).href
        },
        [org, url]
    )

    const submitForm = useCallback(
        (state: string, webhookURL: string, name: string) => {
            if (state && ref.current && formInput.current) {
                const actionUrl = createActionUrl(state)
                ref.current.action = actionUrl
                formInput.current.value = getManifest(webhookURL, name)
                ref.current.submit()
            }
        },
        [createActionUrl, getManifest]
    )

    const createState = useCallback(async () => {
        setError(null)
        try {
            const response = await fetch(`/.auth/githubapp/new-app-state?appName=${name}&webhookURN=${url}`)
            const state: stateResponse = await response.json()
            const webhookURL = new URL(`/.api/webhooks/${state.webhookUUID}`, baseUrl).href
            submitForm(state.state, webhookURL, name)
        } catch (_error) {
            setError(_error)
        }
    }, [submitForm, name, url, baseUrl])

    const handleNameChange = useCallback(
        (event: React.ChangeEvent<HTMLInputElement>) => {
            setName(event.target.value)
        },
        [setName]
    )

    const handleUrlChange = useCallback(
        (event: React.ChangeEvent<HTMLInputElement>) => {
            setUrl(event.target.value)
        },
        [setUrl]
    )

    const handleOrgChange = useCallback(
        (event: React.ChangeEvent<HTMLInputElement>) => {
            setOrg(event.target.value)
        },
        [setOrg]
    )

    return (
        <>
            <PageTitle title="Add GitHubApp" />
            <H2>Add GitHub App</H2>
            <Container>
                {alert}
                {error && <Alert variant="danger">Error creating github app: {error}</Alert>}
                <div className="form-group">
                    <Label className="w-100">
                        <Text alignment="left" className="mb-2">
                            Name
                        </Text>
                        <Input type="text" onChange={handleNameChange} value={name} />
                    </Label>
                </div>
                <div className="form-group">
                    <Label className="w-100">
                        <Text alignment="left" className="mb-2">
                            Base URL
                        </Text>
                        <Input type="text" onChange={handleUrlChange} value={url} />
                    </Label>
                </div>
                <div className="form-group">
                    <Label className="w-100">
                        <Text alignment="left" className="mb-2">
                            Organization (optional)
                        </Text>
                        <Input type="text" onChange={handleOrgChange} value={org} />
                    </Label>
                </div>
                <div className="form-group">
                    <Button variant="primary" onClick={createState}>
                        Create Github App
                    </Button>
                </div>
                {/* eslint-disable-next-line react/forbid-elements */}
                <form ref={ref} method="post">
                    {/* eslint-disable-next-line react/forbid-elements */}
                    <input ref={formInput} name="manifest" onChange={() => {}} hidden={true} />
                </form>
            </Container>
        </>
    )
}
