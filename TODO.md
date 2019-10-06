## TODO

* Add config for all timeouts
* Add config for maxOutstanding, et.c.
* Add to tinysrv
    * auto non-response for docs
* Handle index deletions
* Handle when doc server does not return certain documents
 * Mark as "stale" and continue
* Add context handling to all call flows
* Filter existing documents in document requests
* Add config validation
* Move space init (cfg->db) from db init
* Optionally run in contentless mode
* Add simple in-mem query cache
* Make DocumentManager split too large doc responses
* Add metrics
