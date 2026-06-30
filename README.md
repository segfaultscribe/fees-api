# Fees API

Fees API lets you create a Bill that stays open, add line-items to it facilitating the progressive accrual of fees and review the invoice and line-item charges separately when closing the Bill
# Problem Understanding

Fees API allows a user to create a 'Bill' and add 'Line Items' to it. A bill is like an open tab. The bill receives line-items which are events that incur an amount. The long-running Bill collects these items (progressive accrual) and when closed indicates and makes available the invoice and line-item charges. *ref:* [stripe: Manage bulk invoice line items](https://docs.stripe.com/invoicing/bulk-update-line-item)

A bill also allows support for multiple currencies (currently 'GEL' and 'USD'). However the tricky part is whether a single Bill supports multiple currencies or if a Bill can only be opened for one single currency stream. The approach that Fees API follows is a Bill once set to a currency cannot accept line-items of another currency and the currency has to be passed when creating a Bill. *ref:* [stripe: Multi-currency customers](https://docs.stripe.com/invoicing/multi-currency-customers)

Once a Bill is closed, line-items cannot be added to it.
# Technologies and tools used

Language: Go `go1.26.4`

Workflow: Temporal `v1.7.2 (Server 1.31.1, UI 2.49.1)`

Infrastructure: Encore SDK `v1.57.5`

# Architecture
The proposed architecture uses a Durable execution model made feasible by [Temporal](https://temporal.io/) and uses [Encore](https://encore.dev) infrastructure for setting up the API.

Development follows a Domain Driven Design wherever applicable with focus on Ubiquitous Language and Bounded Context. The development process focuses on building the domain first with the temporal logic depending on the domain and the domain having no other dependencies. Encore wraps the whole app together by forming the infrastructure layer of the application.

The temporal layer builds upon the domain and depends on it. Temporal provides durable execution using workflows and activities. A workflow is a deterministic flow of logic that has guaranteed durability. Which makes it a really promising tool for handling functionality involving money.

```

Domain(data, types, methods) -> Temporal(Workflows{update, query}) -> Encore(API layer, infrastructural concerns, wraps the whole app) -> Client

```

# Running Locally

### Prerequisites

- Go `go1.26.4`
- Temporal CLI`v1.7.2` [Installation Guide](https://docs.temporal.io/cli)
- Encore CLI `v1.57.9` [Installation Guide](https://encore.dev/docs/go/install)

*NOTE:* The Encore CLI version and SDK version are different (Encore SDK `v1.57.5`)

```bash
# Clone the repo
git clone https://github.com/segfaultscribe/fees-api.git

# move into the root folder
cd fees-api

make tidy
# OR
go mod tidy
go mod verify

# These three commands must run simultaneously in separate terminals.

# Terminal 1: Start Temporal Dev Server
temporal server start-dev

# Terminal 2: Start the worker
make worker
# OR
go run cmd/worker/main.go

# Terminal 3: Start the application
make run
# OR
encore run
```

*Note*: Messages to workflows such as Update or Query won't work without a worker.
### Troubleshooting ⚠️

**API calls hang or time out:** ensure the worker process is running 
in a separate terminal before making requests.

**`initService` connection error:** confirm `temporal server start-dev` 
is running and listening on `localhost:7233` before starting the worker 
or the application.

### Code Quality
```bash
# formatting (gofumpt)
make fmt

# linting (golangci-lint)
make lint
```

In order to use formatting and linter, you must have `gofupmt` and `golangci-lint` installed locally. (GitHub actions runs linter when submitting a PR)
# API Reference

All endpoints are served under the `bill` service. Amounts are represented
as decimal strings (e.g. `"12.00"`) at the API boundary to avoid floating
point precision issues; internally they are stored as int64 minor units.

### Create Bill
```
POST /bills
````

Starts a new billing period as a Temporal workflow. The client-supplied
`bill_id` doubles as an idempotency key. Calling this again with the same
`bill_id` returns the bill's initial state without starting a duplicate
workflow.

**Request**
```json
{
  "bill_id": "bill-001",
  "currency": "USD"
}
```

**Response: 200 OK**
```json
{
  "bill_id": "bill-001",
  "currency": "USD",
  "status": "OPEN",
  "created_at": "2026-06-30T19:14:06Z",
  "closed_at": null,
  "line_items": [],
  "total_invoice": "0.00"
}
```

### Add Line Item

```
POST /bills/:billID/line-items
```

Adds a line item to an open bill. The client-supplied `line_id` is used
for idempotency. Sending the same `line_id` again returns the existing line
item rather than adding a duplicate.

**Request**
```json
{
  "line_id": "item-001",
  "description": "cheeseburger",
  "amount": "12.00",
  "currency": "USD"
}
```

**Response: 200 OK**
```json
{
  "line_id": "item-001",
  "description": "cheeseburger",
  "amount": "12.00",
  "currency": "USD"
}
```

**Errors:** 
- `invalid_argument` (malformed request), 
- `not_found` (unknown bill), 
- `failed_precondition` (bill closed or currency mismatch)
### Close Bill
```
POST /bills/:billID/close
```

Closes an open bill and returns the final invoice. Idempotent while the
workflow is running. Calling this again on an already-closed bill (still
within its execution lifetime) returns the same closed state rather than
erroring.

**Response: 200 OK**
```json
{
  "bill_id": "bill-001",
  "currency": "USD",
  "status": "CLOSED",
  "created_at": "2026-06-30T19:14:06Z",
  "closed_at": "2026-06-30T19:20:11Z",
  "line_items": [
    {
      "line_id": "item-001",
      "description": "cheeseburger",
      "amount": "12.00",
      "currency": "USD"
    }
  ],
  "total_invoice": "12.00"
}
```
### Get Bill State

```
GET /bills/:billID
```

Retrieves the current state of a bill, open or closed. Response shape is
identical to Close Bill above.
# Scope

### In Scope

- Create a bill via workflow
- A single Bill supports one of two currencies (USD or GEL)
- Add line items to an open bill(workflow)
- Close a bill and retrieve the invoice and line item charges
- State integrity; rejecting line items on a closed bill
- Currency validation on line item addition to bill

### Out of Scope

- Multi-currency within a single bill
- Database persistence beyond Temporal's retention period
- Automatic bill closure via timer
- Hard limit for number of line items in a bill
- continue-as-new for high-volume line item scenarios
- Authentication and authorization
- Listing all bills

# Process

The core functionalities promised by the Fees API comes down to three major functions

- Create a Bill
- Add Line Items to an Open Bill
- Close the Bill

### Create a Bill

Fees API approaches a Bill quite straightforwardly by treating each Bill as an individual workflow. So creating a Bill starts a temporal workflow with the Bill data stored in an in-memory instance/value.

### Add Line Items to an Open Bill

A Bill is an Open workflow. So adding Line Items to a Bill requires modifying the workflow. In order to interact with the workflow we have three options:
- signal
- query
- update

The earlier approach of interacting with a workflow is to send a signal(async, non-blocking, no response) and then using a query to fetch the workflow state.

With the introduction of update this process has been made a lot easier and hence Fees API uses updates to add Line Items to a Bill. An `Update` is a blocking handler inside the workflow with an ability to send a response when called, which serves our purpose directly. In order to add a line item to the workflow an `Update` will be sent with the payload being the line Item details and the update handler inside the workflow will modify the in-workflow state with the details.
### Close a Bill

A workflow implementation mirrors a normal function with the exception that the entire flow of logic must be deterministic. Any non-deterministic action such as calling an external API, DB Calls etc. must be handed over to an Activity(current implementation of Fees API has no Activity). Since our use case requires a workflow to be a long running process we make use of `workflow.Await()` to make sure the workflow keeps running until it is manually closed.

Closing a bill involves sending an `Update` to the workflow which will update the state of the in-workflow data instance and also satisfy the condition for `workflow.Await()` to return causing the workflow to finally complete its execution.

An additional `Query` handler is also present in the workflow to make feasible the ability to check Bill state even after a workflow is closed. A query has a special feature which makes it work even after a workflow is closed.
# Design Decisions and Tradeoffs

### No Database
Fees API's current implementation does not feature persistence support using a database. This is intentional because the functionalities expected from Fees API can be solved using temporal workflow and workflow features itself. Adding a DB, even though it is normal in real systems, adds additional complexity to our existing use case which makes it hard to defend.

The `Query` handler in the workflow allows retrieval of a Bill's State even after a workflow has completed, which is one of the reasons why a database might be overkill at this stage.

However, considering temporal's retention period which is finite, adding a DB would make sense for a broader objective where you might want to query a Bill's state way long after the workflow has closed which we consider out-of-scope for this implementation.

### Workflow Event Limit 
Temporal maintains its durable execution using an Event History Log. So each workflow is limited to around 51,200 events. (*ref*: [Temporal: Event history limits](https://docs.temporal.io/workflow-execution/event#event-history:~:text=The%20Workflow%20Execution%20is,more%20than%2010000%20Signals.)). This isn't a big problem because temporal provides a `continue-as-new` feature (*ref:* [Temporal: continue-as-new](https://docs.temporal.io/develop/go/workflows/continue-as-new#when:~:text=Use%20Continue%2Das%2DNew%20when%20your%20Workflow%20might%20hit%20Event%20History%20Limits.) where we can start a new workflow and copy the old workflow's state into it and start from 0 events.  

Fees API is currently NOT designed to handle this case because the events in a workflow are determined by the amount of signals, updates or queries that are sent to it and not on how long it runs. Currently, we operate on the assumption that Line Items are added in a reasonable amount with maximum Line Items around ~500. For future scenarios, where the Bill might need to handle a huge number of events, we can consider using the `continue-as-new` operation.
### Idempotency
It is important for a finance system to be very strict about idempotency, however, considering scope and simplicity Fees API follows a simplistic idempotency implementation.

The BillID and LineID are client-supplied and hence assumed to be an idempotent
key along with being the internal identification fields. This pattern offloads the generation to client hence actively keeps idempotency without the complication of a header based idempotency key + ID pattern (like stripe) for data objects.

When adding a Line Item to a Bill, the update handler in the workflow checks whether a `LineID` already exists in the bill's `LineItems` map before adding. If found, it returns the existing item unchanged rather than inserting a duplicate, making the operation safe to retry without side effects.
### Currency
Currency representation in Fees API is handled by accepting currency as a decimal string and parsing it into its minor unit as an `int64`. So `$23.32` becomes `2332 cents` in the application and on response is converted back.
### Domain models
Line Items that are added to a Bill can have amount 0. This is done to allow addition of free items, gifts or items of such nature. The items might have a description but no associated cost.

`LineItem` cannot exist independently and can exist only as a part of a Bill.
### Infrastructure
Fees API does not have a `.env` file or a config file which is deliberate. The correct pattern would be to set up a config file and use encore to generate a `.cue` file which will then populate config values across the service. The complexity associated with handling .cue and the small number of config required eventually led to hard-coded values for the demo. However it is important to note that on deployment or further development setting up config would be the right move.

No Docker / Docker Compose for Temporal. `temporal server start-dev` is the documented, lightweight standard for local dev. Docker would add setup complexity without demonstrating anything relevant to the current scope. 
### Endpoint
Proper REST API semantics state that the endpoints must end or contain only noun or plural form of nouns, however `POST /bills/:billId/close` in Fees API is deliberate. Sometimes especially in complicated setups using a verb makes more sense that fiddling with HTTP methods. This is similar to some of stripe's API Design.
# Deferred/Future features
`ListBills` endpoint not implemented: Temporal's visibility API (`ListWorkflow`) doesn't expose workflow-internal state like currency or computed totals without either custom Search Attributes or complicates looped queries. The lack of a database doesn't help either.

Initial workflow testing was done using temporal's `testsuite` which provided a convenient sandbox environment to run and test workflows. However complication when testing Updates specifically made it too time consuming. Hence testing workflows was reverted to using go's in-built `testing` framework. However with more time `testsuite` can be used for better testing capabilities.

Adding configuration using `.cue` files and encore is much relevant to when the system rolls out to deployment.

For more operation on data in the system, persistence via a database (encore provides managed cloud PostgreSQL and local PostgreSQL) will be a great addition.