# Reports Server

Reports server provides a scalable solution for storing policy reports and cluster policy reports. It moves reports out of etcd and stores them in a PostgreSQL database instance.

## Why reports server?

It is desirable to move reports out of etcd for several reasons:

Why leave etcd:

- The etcd database currently has a maximum size of 8GB. Reports tend to be relatively large objects, but even with very small reports, the etcd capacity can be easily reached in larger clusters with many report producers.
- Under heavy report activity (e.g. cluster churn, scanning, analytical processes, etc.), the volume of data being written and retrieved by etcd requires the API server to buffer large amounts of data. This compounds existing cluster issues and can cascade into complete API unavailability.
- CAP guarantees are not required for reports, which, at present, are understood to be ephemeral data that will be re-created if deleted.
- Philosophically, report data is analytical in nature and should not be stored in the transactional database.

Benefits of reports server:

- Alleviation of the etcd + API server load and capacity limitations.
- Common report consumer workflows can be more efficient.
    - Report consumers are often analytical and/or operate on aggregate data. The API is not designed to efficiently handle, for example, a query for all reports where containing a vulnerability with a CVSS severity of 8.0 or above. To perform such a query, a report consumer must retrieve and parse all of the reports. Retrieving a large volume of reports, especially with multiple simultaneous consumers, leads to the performance issues described previously.
    - With reports stored in a relational database, report consumers could instead query the underlying database directly, using more robust query syntax.
    - This would improve the implementation of, or even replace the need for, certain exporters, and enable new reporting use cases.

## Installation

See [INSTALL.md](/docs/INSTALL.md)
