# Temporal Go SDK - Advanced Utilities

This repository houses advanced utilities as I work on them. They are not official and there are no API compatibility
guarantees. Instead, the code may be lifted/reused at will.

Usable utilities:

* [temporalproto](temporalproto)
  [![Go Reference](https://pkg.go.dev/badge/github.com/cretz/temporal-sdk-go-advanced/temporalproto.svg)](https://pkg.go.dev/github.com/cretz/temporal-sdk-go-advanced/temporalproto)
  - Protobuf generator for Temporal workflows and activities

TODO(cretz): This doc, including:
* How non-determinism check is done and caveats/alternatives
* How read-only check is done and caveats/alternatives
* Continue as new burdens and serialize size
* Deadlock detection