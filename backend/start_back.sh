#!/bin/bash

donutbin --service_hostport="front-envoy:80" --tracer_type=${TRACER} & 
