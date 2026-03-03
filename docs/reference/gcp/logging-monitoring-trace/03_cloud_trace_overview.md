# Cloud Trace overview  |  Google Cloud Documentation

Source: https://docs.cloud.google.com/trace/docs/overview

* [ Home ](https://docs.cloud.google.com/)
  * [ Documentation ](https://docs.cloud.google.com/docs)
  * [ Observability ](https://docs.cloud.google.com/docs/observability)
  * [ Cloud Trace ](https://docs.cloud.google.com/trace/docs)
  * [ Guides ](https://docs.cloud.google.com/trace/docs/overview)



Send feedback 

#  Cloud Trace overview Stay organized with collections  Save and categorize content based on your preferences. 

Cloud Trace, a distributed tracing system for Google Cloud, helps you understand how long it takes your application to handle incoming requests from users or other applications, and how long it takes to complete operations like RPC calls performed when handling the requests. Trace can also help you when you are developing a service or troubleshooting a failure. For example, it can help you understand how requests are processed in a complicated microservices architecture, and it might help you identify which logs to examine.

Because Trace receives latency data from some Google Cloud services, such as [App Engine](https://cloud.google.com/appengine/docs), and from applications instrumented with the [Cloud Trace API](https://cloud.google.com/trace/api), it can help you answer the following questions:

  * How long does it take my application to handle a given request?
  * Why is it taking my application so long to handle a request?
  * Why do some of my requests take longer than others?
  * What is the overall latency of requests to my application?
  * Has latency for my application increased or decreased over time?
  * What can I do to reduce application latency?
  * What are my application's dependencies?



If you're curious about how you can use Trace to help you manage your applications, then read the blog [Troubleshooting distributed applications: Using traces and logs together for root-cause analysis](https://cloud.google.com/blog/products/devops-sre/using-cloud-trace-and-cloud-logging-for-root-cause-analysis).

For information about profiling your application, see [Cloud Profiler](https://cloud.google.com/profiler/docs).

## Environment support

Trace runs on Linux in the following environments:

  * [Compute Engine](https://cloud.google.com/compute/docs)
  * [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/docs)
  * [Apigee](https://cloud.google.com/apigee/docs/api-platform/develop/enabling-distributed-trace) (Public Preview)
  * [App Engine flexible environment](https://cloud.google.com/appengine/docs/flexible)
  * [App Engine standard environment](https://cloud.google.com/appengine/docs/standard)
  * [Cloud Run](https://cloud.google.com/run/docs/trace)
  * [Cloud Service Mesh](https://cloud.google.com/service-mesh/docs/observability/accessing-traces)
  * [Cloud SQL query insights](https://cloud.google.com/sql/docs/mysql/using-query-insights)
  * Non-Google Cloud environments



Trace provides client libraries for instrumenting your application to capture trace information. For per-language setup instructions, see [Instrument for Trace](https://cloud.google.com/trace/docs/setup).

## Configurations with automatic tracing

Some configurations result in automatic capture of trace data:

  * App Engine standard environment

Java 8, Python 2, and PHP 5 applications don't need to use the Trace client libraries. These runtimes automatically send latency data to Trace for requests to application URIs. The requests include latency data for round-trip RPC calls to App Engine services. Trace works with all App Engine Admin APIs, with the exception of Cloud SQL.

  * Cloud Run functions and Cloud Run

For incoming and outgoing HTTP requests, latency data is automatically sent to Trace.




## Language support

**Note:** You can instrument your application so that it collects application-specific information. Several open-source instrumentation frameworks let you collect metrics, logs, and traces from your application and send that data to any vendor, including Google Cloud. To instrument your application, we recommend that you use a vendor-neutral instrumentation framework that is open source, such as [OpenTelemetry](https://opentelemetry.io/), instead of vendor- and product-specific APIs or client libraries. 

For information about instrumenting your applications by using vendor-neutral instrumentation frameworks, see [ Instrumentation and observability](https://cloud.google.com/stackdriver/docs/instrumentation/overview). 

The following table summarizes the availability of Trace client libraries and of [OpenTelemetry](https://opentelemetry.io/) libraries for which there is an exporter to Trace.

Language | Client library   
available | OpenTelemetry   
lib/exporter available  
---|---|---  
[C++](https://cloud.google.com/trace/docs/setup/cpp-ot) | Yes | Yes  
C# ASP.NET Core | Yes | No  
C# ASP.NET | Yes | No  
[Go](https://cloud.google.com/trace/docs/setup/go-ot) | Yes | Yes  
[Java](https://cloud.google.com/trace/docs/setup/java-ot) | Yes | Yes  
[Node.js](https://cloud.google.com/trace/docs/setup/nodejs-ot) | Yes | Yes  
[PHP](https://cloud.google.com/php/docs/reference/cloud-trace/latest) | Yes | No  
[Python](https://cloud.google.com/trace/docs/setup/python-ot) | Yes | Yes  
Ruby | Yes | Yes  
  
[OpenTelemetry](https://opentelemetry.io/) libraries are simpler to use than the Trace client libraries because they hide some of the complexity of the corresponding Trace API. For instrumentation recommendations, see [Choose an instrumentation approach](https://cloud.google.com/stackdriver/docs/instrumentation/choose-approach).

## Components

Trace consists of a tracing client, which collects _traces_ and sends them to your Google Cloud project. You can then use the Google Cloud console to view and analyze the data collected by the agent. For information about the data model, see [Traces and spans](https://cloud.google.com/trace/docs/traces-and-spans).

### Tracing client

If an OpenTelemetry library is available for your programming language, you can simplify the process of creating and sending trace data by using [OpenTelemetry](https://opentelemetry.io/). In addition to being simpler to use, OpenTelemetry implements batching which might improve performance.

If an OpenTelemetry library doesn't exist, then instrument your code by importing the Trace SDK library and by using the Cloud Trace API. The Cloud Trace API sends trace data to your Google Cloud project.

### Tracing interface

You can view and analyze your trace data in near real-time in the Trace interface.

To view and analyze your span data, you can use the **Trace Explorer** and **Log Analytics** pages in the Google Cloud console:

  * **Trace Explorer** : Displays aggregate information about your trace data and lets you examine individual traces in detail. The aggregated latency data is shown on a heatmap, which you can explore with your pointer. To restrict which data is displayed, you can add filters. This page also lets you view and explore individual spans and traces:

    * For information about how to view trace data stored in multiple projects, see [Create and manage trace scope](https://cloud.google.com/trace/docs/trace-scope/create-and-manage).
    * For information about filtering and viewing your trace data, see [Find and explore traces](https://cloud.google.com/trace/docs/finding-traces).
  * **Log Analytics** : This page lets you run queries that perform an aggregate analysis of your spans by using SQL. Your SQL queries can also join your trace and log data. You can view the results of your query in tabular form or with charts. If you create a linked dataset, then you can use [BigQuery](https://cloud.google.com/bigquery/docs/introduction) to analyze your spans. For more information, see [Query and analyze traces](https://cloud.google.com/trace/docs/analytics).




## VPC Service Controls support

Trace is a VPC Service Controls supported service. The Trace service name is `cloudtrace.googleapis.com`. Any VPC Service Controls restrictions that you create for the Trace service apply only to that service. Those restrictions don't apply to any other services, including those like the [`telemetry.googleapis.com` service](https://cloud.google.com/stackdriver/docs/reference/telemetry/overview), which can also ingest trace data.

For more information, see the following:

  * [VPC Service Controls documentation](https://cloud.google.com/vpc-service-controls/docs).
  * [Supported products and limitations](https://cloud.google.com/vpc-service-controls/docs/supported-products).



## Cloud Trace and data residency

If you are using [Assured Workloads](https://cloud.google.com/security/products/assured-workloads) because you have data-residency or [Impact Level 4 (IL4)](https://cloud.google.com/security/compliance/disa) requirements, then don't use the Cloud Trace API to send trace spans.

## Pricing

To learn about pricing for Cloud Trace, see the [Google Cloud Observability pricing](https://cloud.google.com/products/observability/pricing) page.

## What's next

  * Try the [Quickstart](https://cloud.google.com/trace/docs/trace-app-latency).

  * For information about quotas and limits, see [Quotas and limits](https://cloud.google.com/trace/docs/quotas).

  * Read our resources about [DevOps](https://cloud.google.com/devops) and explore the [DevOps Research and Assessment](https://dora.dev/) research program.




Send feedback 

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2026-02-26 UTC.

Need to tell us more?  [[["Easy to understand","easyToUnderstand","thumb-up"],["Solved my problem","solvedMyProblem","thumb-up"],["Other","otherUp","thumb-up"]],[["Hard to understand","hardToUnderstand","thumb-down"],["Incorrect information or sample code","incorrectInformationOrSampleCode","thumb-down"],["Missing the information/samples I need","missingTheInformationSamplesINeed","thumb-down"],["Other","otherDown","thumb-down"]],["Last updated 2026-02-26 UTC."],[],[]]
