# orb-cli

`orb-cli` is a command-line tool for [Orb](https://withorb.com). It runs a
local, read-only MCP server.

## Development

This repository has a [devenv](https://devenv.sh) environment. It provides the
Go development environment.

You can also use a compatible Go installation.

## Installation

Install the command from this repository:

```sh
go install ./cmd/orb-cli
```

Go installs the command in `$(go env GOPATH)/bin`. Add this directory to your
`PATH` if needed.

## Usage

### Install for Claude Code

```sh
claude mcp add --transport stdio \
  --env ORB_LIVE_API_KEY="$ORB_LIVE_API_KEY" \
  --env ORB_TEST_API_KEY="$ORB_TEST_API_KEY" \
  orb -- orb-cli mcp
```

Set both key variables in your shell before you run this command.

### Install with add-mcp

Use `add-mcp` to add the server to multiple agent harnesses:

```sh
npx add-mcp "orb-cli mcp" \
  --name orb \
  --yes \
  --env "ORB_LIVE_API_KEY=$ORB_LIVE_API_KEY" \
  --env "ORB_TEST_API_KEY=$ORB_TEST_API_KEY"
```

Set both key variables in your shell before you run this command.

## MCP tools

The server provides these read-only tools:

- `orb_list_customers`
- `orb_list_subscriptions`
- `orb_list_plans`
- `orb_list_invoices`
- `orb_list_invoice_summaries`
- `orb_get_customer`
- `orb_get_subscription`
- `orb_get_plan`
- `orb_get_invoice`
