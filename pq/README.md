# pq

`pq` (short for '*P*rometheus *Q*uery') is a cli tool for interacting with 
raw prometheus metric endpoints. This is particularly useful if, for some reason, you do 
not have a prometheus backend set-up already and you want to run ad-hoc queries against 
some prometheus metrics. 

With the `pq` executable binary, you can execute arbitrary promql queries against an http endpoint which returns valid prometheus metric data. This allows you to query for specific metrics, like so:


```bash
$ pq metrics -q apiserver_request_count
```

In line with Kubernetes conventions, you can also specify your returned data format:

```bash
$ pq metrics -q apiserver_request_count -oyaml
```

If you need more complicated queries, pq allows you to evaluate arbitrarily complex promql queries:

```bash
$ pq metrics -q 'sum(apiserver_request_count{verb="POST"}) by (resource, code)'
```

If you want to run pq interactively, you can! PQ can continuously scrape a prometheus endpoint 
and store the data in memory. You can enable this by using the `--continuous` (or `-c`) flag.

## Architecture 

PQ stores scraped metric data in memory. This means that if you run this cli in 
continuous-mode, you will eventually run out of memory. This is by design, since
this tool isn't meant to replace a prometheus server backend. Though not implemented yet, we do envision having a circular buffer of sorts, with the ability to customize scrape period and in-memory retention. 

### Building from source

We use a standard go build to build from source code. You will want to move the built binary somewhere in your
path. Assuming you have a `~/bin` directory in your `PATH`, the build would look like this:
 
```bash
go build pq.go && cp pq ~/bin 
```