# listpages

The `listpages` generator creates paginated variants of AWS Go SDK functions that return collections of objects where the SDK does not define them. It should typically be called using [`go generate`](https://golang.org/cmd/go/#hdr-Generate_Go_files_by_processing_source).

For example, the EC2 API defines both [`DescribeInstancesPages`](https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstancesPages) and  [`DescribeInstances`](https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstances), whereas the CloudWatch Events API defines only [`ListEventBuses`](https://docs.aws.amazon.com/sdk-for-go/api/service/cloudwatchevents/#CloudWatchEvents.ListEventBuses).

The `listpages` executable is called as follows:

```console
$ go run main.go -ListOps <function-name>[,<function-name>]
```

* `<function-name>`: Name of a function to wrap

Optional Flags:

* `-Paginator`: Name of the pagination token field (default `NextToken`)
* `-Export`: Whether to export the generated functions

To use with `go generate`, add the following directive to a Go file

```go
//go:generate go run <relative-path-to-generators>/generate/listpages/main.go -ListOps=<comma-separated-list-of-functions>
```

For example, in the file `internal/service/cloudwatchevents/generate.go`

```go
//go:generate go run -tags generate ../../generate/listpages/main.go -ListOps=ListEventBuses,ListRules,ListTargetsByRule

package cloudwatchevents
```

generates the file `internal/service/cloudwatchevents/list_pages_gen.go` with the functions `listEventBusesPages`, `listRulesPages`, and `listTargetsByRulePages` as well as their `...WithContext` equivalents.
