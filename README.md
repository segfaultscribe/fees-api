# Fees API

Fees API lets you create a Bill that stays open, add line-items to it facilitating the progressive accrual of fees and review the invoice and line-item charges separately when closing the Bill

# Problem Understanding

Fees API allows a user to create a 'Bill' and add 'Line Items' to it. A bill is like an open tab. The bill receives line-items which are events that incur an amount. The long-running Bill collects these items (progressive accrual) and when closed indicates and makes available the invoice and line-item charges. *ref:* [stripe: Manage bulk invoice line items](https://docs.stripe.com/invoicing/bulk-update-line-item)

A bill also allows support for multiple currencies (currently 'GEL' and 'USD'). However the tricky part is whether a single Bill supports multiple currencies or if a Bill can only be opened for one single currency stream. The approach that Fees API follows is a Bill once set to a currency cannot accept line-items of another currency and the currency has to be passed when creating a Bill. *ref:* [stripe: Multi-currency customers](https://docs.stripe.com/invoicing/multi-currency-customers)

Once a Bill is closed, line-items cannot be added to it.

# Technologies and tools used
Language: Go `go1.26.4`

Workflow: Temporal `v1.7.2 (Server 1.31.1, UI 2.49.1)`

Infrastructure: Encore `v1.57.9`
# Architecture

The proposed architecture uses a Durable execution model made feasible by [Temporal](https://temporal.io/) and uses [Encore](https://encore.dev) infrastructure for setting up the API. 

Development follows a Domain Driven Design wherever applicable with focus on  Ubiquitous Language and Bounded Context. The development process focuses on building the domain first with the temporal logic depending on the domain and the domain having no other dependencies. Encore wraps the whole app together by forming the infrastructure layer of the application.

The temporal layer builds upon the domain and depends on it. Temporal provides durable execution using workflows and activities. A workflow is a deterministic flow of logic that has guaranteed durability. Which makes it a really promising tool for handling functionality involving money.

```
Domain(data, types, methods) -> Temporal(Workflows{update, query}) -> Encore(API layer, infrastructural concerns, wraps the whole app) -> Client
```
# Running Locally
### Prerequisites 
- Go `go1.26.4`
- Temporal CLI`v1.7.2` [Installation Guide](https://docs.temporal.io/cli)
- Encore `v1.57.9` [Installation Guide](https://encore.dev/docs/ts/install)

```bash
# Start Temporal Dev Server 
temporal server start-dev 
# Start the Application 
encore run 
# Run Tests 
make test 
```

> Full setup instructions will be completed at v1.0-feeapi

# Scope 
### In Scope 
- Create a bill via workflow
- A single Bill supports one of two currency (USD or GEL) 
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
The core functionalities promised by the Fees-API comes down to three major functions
- Create a Bill
- Add Line Items to an Open Bill
- Close the Bill
### Create a Bill
Fees-API approaches a Bill quite straightforwardly by treating each Bill as an individual workflow. So creating a Bill starts a temporal workflow with the Bill data stored in an in-memory instance/value.

### Add Line Items to an Open Bill
A Bill is an Open workflow. So adding Line Items to a Bill requires modifying the workflow. In order to interact with the workflow we have three options:
- signal
- query
- update

The earlier approach of interacting with a workflow is to send a signal(async, non-blocking, no response) and then using a query to fetch the workflow state. 

With the introduction of update this process has been made a lot easier and hence Fees-API uses updates to add Line Items to a Bill. An `Update` is a blocking handler inside the workflow with an ability to send a response when called, which serves our purpose directly. In order to add a line item to the workflow an `Update` will be sent with the payload being the line Item details and the update handler inside the workflow will modify the in-workflow state with the details.

### Close a Bill
A workflow implementation mirrors a normal function with the exception that the entire flow of logic must be deterministic. Any non-deterministic action such as calling an external API, DB Calls etc. must be handed over to an Activity. Since our use case requires a workflow to be a long running process we make use of `workflow.Await()` to make sure the workflow keeps running until it is manually closed. 

Closing a bill involves sending an `Update` to the workflow which will update the state of the in-workflow data instance and also satisfy the condition for `workflow.Await()` to return causing the workflow to finally complete it's execution.

An additional `Query` handler is also present in the workflow to make feasible the ability to check Bill state even after a workflow is closed. A query has a special feature which makes it work even after a workflow is closed. 

# Concerns
**No Database**: Fees-APIs current implementation does not feature persistence support using a database. This is intentional because the functionalities expected from Fees-API can be solved using temporal workflow and workflow features itself. Adding a DB, even though it is normal in real systems, adds additional complexity to our existing use case which makes it hard to defend.
  
The `Query` handler in the workflow allows retrieval of a Bill's State even after a workflow has completed, which is one of the reasons why a database might be overkill at this stage. 

However, considering temporal's retention period which is finite, adding a DB would make sense for a broader objective where you might want to query a Bill's state way long after the workflow has closed which we consider out-of-scope for this implementation.

**Workflow Event Limit**: Temporal maintains it's durable execution using an Event History Log. So each workflow is limited to around 51,200 events. (*ref*: [Temporal: Event history limits](https://docs.temporal.io/workflow-execution/event#event-history:~:text=The%20Workflow%20Execution%20is,more%20than%2010000%20Signals.)). This isn't a big problem because temporal provides a `continue-as-new` feature (*ref:* [Temporal: continue-as-new](https://docs.temporal.io/develop/go/workflows/continue-as-new#when:~:text=Use%20Continue%2Das%2DNew%20when%20your%20Workflow%20might%20hit%20Event%20History%20Limits.) where we can start a new workflow and copy the old workflow's state into it and start from 0 events.   

Fees-API is currently NOT designed to handle this case because the events in a workflow are determined by the amount of signals, updates or queries that are sent to it and not on how long it runs. Currently, we operate on the assumption that Line Items are added in a reasonable amount with maximum Line Items around ~500. For future scenarios, where the Bill might need to handle a huge number of events, we can consider using the `continue-as-new` operation.
# Development Phases

### v0.1.0-domain 
- domain types, structures and essential methods. 
- Bill, Line Item, Currency types. 
- Methods for Bill, Line Item and Currency
- Full unit test coverage.

### v0.2.0-temporal 
- Workflow using the domain. 
- Update handlers for add-line-item and close-bill.
- Query handler for invoice retrieval. 
- Workflow testing.

### v0.3.0-encore API Layer
- HTTP endpoints exposed via Encore.
- temporal client, workflow and worker setup  
- Full end-to-end flow test.
- polish, final README.md