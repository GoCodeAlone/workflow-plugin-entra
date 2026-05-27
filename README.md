# workflow-plugin-entra

Microsoft Entra ID management provider plugin for Workflow. It uses the official
Microsoft Graph Go SDK and Azure Identity SDK.

## Capabilities

- `entra.provider` module using Entra client credentials with Microsoft Graph
- Auth provider descriptor step for admin catalog integration
- User create/read/list/update/delete steps
- Group create/read/list and direct membership management steps
- Application registration create/read/list/update/delete steps
- Service principal and directory role list steps

The descriptor advertises only capabilities backed by the plugin's concrete
management steps.

## Install

```sh
wfctl plugin install workflow-plugin-entra
```

## License

MIT
