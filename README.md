# Letarette

If all you need is a scalable, simple, understandable search engine - look no further.
Letarette is easy to set up and integrate with your data.

If you need customizable stemming, suggestions, ... Letarette might not be for you.
There are several options, including Elasticsearch, ...  .

You might be surprised.

## Getting started

Worker

Document manager

## Details

### Topics

The top nats topic is configurable and defaults to "leta".
The following sub topics are used:

- leta.q:
    >Query request. Sent from search client. Fastest worker to respond wins.
- leta.status:
    >Worker status request. All workers respond with their individual status.
- leta.index.status:
    >Index status request. Sent from worker to document master to get current state of the source.
- leta.index.request:
    >Index update request. Sent from worker to document master to get a list of updates.
- leta.document.request:
    >Document request. Sent from worker to get documents. Served by other workers or the master.
- leta.document.update:
    >Sent in response to document requests.

## TODO

- [ ] Snowball stemmer
- [ ] Bulk initialization
- [ ] NATS Authentication
