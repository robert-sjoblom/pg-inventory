# pg-inventory

## Overview
This project is an in-house solution (for more details, see history) to provide developers visibility into PostgreSQL instances. Developers have no access to production deployments, but still need to see data (such as query statistics or `pg_stat_activity`/`pg_locks`). This catalog exposes that data.

It can handle PostgreSQL HA setups with primary and replica instances, and saves data in a timeseries database (TimescaleDB).

Originally built for VM-based deployments, it "should" do fine as a k8s sidecar (the extractor, that is). Untested, though!

### Requirements
- PostgreSQL 14 or higher

## History
This all started as a hackathon project, written in Python. It was then rewritten in Rust for improved performance and safety (as well as improved development speed). I am now releasing it as an open-source variant, written in Go. For various reasons (mostly the fact that the rust version is tightly coupled to the in-house deployments of PG), a rewrite for the open-source version is in order. Since I (the author and the rust dev, but not the original dev) want to learn Go this makes sense to me. Additionally, we're also moving from a JSON response to gRPC communication.

## Architecture
Originally, I wrote a long text on what the PostgreSQL deploys look like where I'm at, but that doesn't matter here. The only thing you need to know is that traffic only goes one way: collectors can reach extractors, but extractors cannot reach collectors. It's pull, not push. An unfortunate side-effect of our in-house setup (multiple AZs, VLANs and etc).

### extractor
Simple gRPC server that exposes the data we want from a given PostgreSQL server.

TODO: specify user access levels for the extractor.

### collector
The collector reaches out to a specific host and collects data from that host. It then inserts that data into the central database.

### api-service
A simple API service, exposes the data collected for any given frontend to use/display as they see fit.

## Prerequisites
TODO!

## Installation
TODO!

## Configuration
TODO!

### extractor Configuration

### collector Configuration

### api-service Configuration
