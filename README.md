# nel-collector

This is a small agent that listens for HTTP on a port and logs
[NEL](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Network_Error_Logging)
responses to a database.  This can be used to collect client-side HTTP
metrics from browsers, including HTTP errors and timing metrics.

This mostly only works with Chrome-family browsers today; Firefox has
support but it's disabled by default.

This is still a work in progress.
