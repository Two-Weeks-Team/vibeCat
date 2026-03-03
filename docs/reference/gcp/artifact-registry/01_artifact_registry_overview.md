# Artifact Registry overview  |  Google Cloud Documentation

Source: https://docs.cloud.google.com/artifact-registry/docs/overview

* [ Home ](https://docs.cloud.google.com/)
  * [ Documentation ](https://docs.cloud.google.com/docs)
  * [ Application development ](https://docs.cloud.google.com/docs/application-development)
  * [ Artifact Registry ](https://docs.cloud.google.com/artifact-registry/docs)
  * [ Guides ](https://docs.cloud.google.com/artifact-registry/docs/overview)



Send feedback 

#  Artifact Registry overview Stay organized with collections  Save and categorize content based on your preferences. 

Artifact Registry lets you centrally store artifacts and build dependencies as part of an integrated Google Cloud experience.

## Introduction

Artifact Registry provides a single location for storing and managing your packages and Docker container images. You can:

  * Integrate Artifact Registry with Google Cloud [CI/CD services](https://cloud.google.com/blog/topics/developers-practitioners/devops-and-cicd-google-cloud-explained) or your existing CI/CD tools. 
    * Store artifacts from [Cloud Build](https://cloud.google.com/build/docs).
    * Deploy artifacts to Google Cloud runtimes, including [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/docs), [Cloud Run](https://cloud.google.com/run/docs), [Compute Engine](https://cloud.google.com/compute/docs), and [App Engine flexible environment](https://cloud.google.com/appengine/docs/flexible).
    * Identity and Access Management provides consistent credentials and access control.
  * Protect your software supply chain. 
    * Manage container metadata and scan for container vulnerabilities with [Artifact Analysis](https://cloud.google.com/artifact-analysis/docs).
  * Protect repositories in a [VPC Service Controls](https://cloud.google.com/vpc-service-controls/docs/overview) security perimeter.
  * Create multiple regional repositories within a single Google Cloud project. Group images by team or development stage and control access at the repository level.

Artifact Registry integrates with Cloud Build and other continuous delivery and continuous integration systems to store packages from your builds. You can also store trusted dependencies that you use for builds and deployments. 

## Dependency management

Protecting your software supply chain goes beyond using specific tools. The processes and practices you use to develop, build, and run your software also impact the integrity of your software. To learn more about best practices for dependencies, see [Dependency management](https://cloud.google.com/software-supply-chain-security/docs/dependencies)

## Software supply chain security

Google Cloud provides a comprehensive and modular set of capabilities and tools that your developers, DevOps, and security teams can use to improve the security posture of your software supply chain.

Artifact Registry provides:

  * Remote repositories to cache dependencies from upstream public sources so that you have greater control over them and can scan them for vulnerabilities, build provenance, and other dependency information.
  * Virtual repositories to group remote and private repositories behind a single endpoint. Set a priority on each repository to control search order when downloading or installing an artifact.



You can view security insights about your security posture, build artifacts, and dependencies in Google Cloud console dashboards within Cloud Build, Cloud Run, and GKE.

## Artifact Registry and Container Registry

Artifact Registry expands on the capabilities of Container Registry and is the recommended container registry for Google Cloud. If you currently use Container Registry, learn about [transitioning from Container Registry](https://cloud.google.com/artifact-registry/docs/transition-from-gcr) to take advantage of new and improved features. 

## What's next

  * [Docker quickstart](https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images)
  * [Go quickstart](https://cloud.google.com/artifact-registry/docs/go/store-go)
  * [Helm quickstart](https://cloud.google.com/artifact-registry/docs/helm/store-helm-charts)
  * [Java quickstart](https://cloud.google.com/artifact-registry/docs/java/store-java)
  * [Node.js quickstart](https://cloud.google.com/artifact-registry/docs/nodejs/store-nodejs)
  * [Python quickstart](https://cloud.google.com/artifact-registry/docs/python/store-python)
  * [Ruby quickstart](https://cloud.google.com/artifact-registry/docs/ruby/quickstart)



Send feedback 

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2026-02-25 UTC.

Need to tell us more?  [[["Easy to understand","easyToUnderstand","thumb-up"],["Solved my problem","solvedMyProblem","thumb-up"],["Other","otherUp","thumb-up"]],[["Hard to understand","hardToUnderstand","thumb-down"],["Incorrect information or sample code","incorrectInformationOrSampleCode","thumb-down"],["Missing the information/samples I need","missingTheInformationSamplesINeed","thumb-down"],["Other","otherDown","thumb-down"]],["Last updated 2026-02-25 UTC."],[],[]]
