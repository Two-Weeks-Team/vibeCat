# Overview of Cloud Build  |  Google Cloud Documentation

Source: https://docs.cloud.google.com/build/docs/overview

* [ Home ](https://docs.cloud.google.com/)
  * [ Documentation ](https://docs.cloud.google.com/docs)
  * [ Application development ](https://docs.cloud.google.com/docs/application-development)
  * [ Cloud Build ](https://docs.cloud.google.com/build/docs)
  * [ Guides ](https://docs.cloud.google.com/build/docs/build-push-docker-image)



Send feedback 

#  Overview of Cloud Build Stay organized with collections  Save and categorize content based on your preferences. 

Cloud Build is a service that executes your builds on Google Cloud.

Cloud Build can import source code from a variety of repositories or cloud storage spaces, execute a build to your specifications, and produce artifacts such as Docker containers or Java archives. You can also use Cloud Build to help protect your software supply chain. Cloud Build features meet the requirements of Supply chain Levels for Software Artifacts (SLSA) level 3. For guidance on protecting your build processes, see [Safeguard builds](https://cloud.google.com/software-supply-chain-security/docs/safeguard-builds). 

## Build configuration and build steps

You can write a [build config](https://cloud.google.com/build/docs/build-config) to provide instructions to Cloud Build on what tasks to perform. You can configure builds to fetch dependencies, run unit tests, static analyses, and integration tests, and create artifacts with build tools such as docker, gradle, maven, bazel, and gulp.

Cloud Build executes your build as a series of build steps, where each build step is run in a Docker container. Executing build steps is analogous to executing commands in a script.

You can either use the build steps provided by Cloud Build and the Cloud Build community, or write your own custom build steps:

  * **Build steps provided by Cloud Build** : Cloud Build has published a set of [supported open-source build steps](https://github.com/GoogleCloudPlatform/cloud-builders) for common languages and tasks.
  * **Community-contributed build steps** : The Cloud Build user community has provided open-source [build steps](https://github.com/GoogleCloudPlatform/cloud-builders-community).
  * **Custom build steps** : You can [ create your own build steps](https://cloud.google.com/build/docs/create-custom-build-steps) for use in your builds.



Each build step is run with its container attached to a local Docker network named `cloudbuild`. This allows build steps to communicate with each other and share data. For more information on the `cloudbuild` network, see [Cloud Build network](https://cloud.google.com/build/docs/build-config-file-schema#network).

You can use standard [Docker Hub](https://hub.docker.com/) images in Cloud Build, such as [Ubuntu](https://hub.docker.com/_/ubuntu/) and [Gradle](https://hub.docker.com/_/gradle/).

## Starting builds

You can [manually start builds](https://cloud.google.com/build/docs/running-builds/start-build-manually) in Cloud Build using the Google Cloud CLI or the [Cloud Build API](https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/create), or use [Cloud Build's build triggers](https://cloud.google.com/build/docs/triggers) to create an automated continuous integration/continuous delivery (CI/CD) workflow that starts new builds in response to code changes. You can integrate build triggers with many code repositories, including [Cloud Source Repositories](https://cloud.google.com/source-repositories/docs), GitHub, and Bitbucket. 

## Viewing build results

You can view your build results using the gcloud CLI, the [Cloud Build API](https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/list) or use the **Build History** page in the Cloud Build section in Google Cloud console, which displays details and logs for every build Cloud Build executes. For instructions see [Viewing Build Results](https://cloud.google.com/build/docs/view-build-results).

## How builds work

The following steps describe, in general, the lifecycle of a Cloud Build build:

  1. Prepare your application code and any needed assets.
  2. Create a build config file in YAML or JSON format, which contains instructions for Cloud Build.
  3. Submit the build to Cloud Build.
  4. Cloud Build executes your build based on the build config you provided.
  5. If applicable, any built artifacts are pushed to [Artifact Registry](https://cloud.google.com/artifact-registry/docs).



### Docker

Cloud Build uses [Docker](https://docker.com/) to execute builds. For each build step, Cloud Build executes a Docker container as an instance of `docker run`. Currently, Cloud Build is running Docker engine version 20.10.24.

## Cloud Build interfaces

You can use Cloud Build with the Google Cloud console, `gcloud` command-line tool, or Cloud Build's REST API.

In the Google Cloud console, you can [view the Cloud Build build results in the **Build History** page](https://cloud.google.com/build/docs/view-build-results), and [automate builds in **Build Triggers**](https://cloud.google.com/build/docs/running-builds/automate-builds). 

You can use the gcloud CLI to [create and manage builds](https://cloud.google.com/sdk/gcloud/reference/builds). You can run commands to perform tasks such as [submitting a build](https://cloud.google.com/sdk/gcloud/reference/builds/submit), [listing builds](https://cloud.google.com/sdk/gcloud/reference/builds/list), and [canceling a build](https://cloud.google.com/sdk/gcloud/reference/builds/cancel).

You can request builds using the [Cloud Build REST API](https://cloud.google.com/build/docs/api/reference/rest).

As with other Cloud Platform APIs, you must authorize access using [OAuth2](https://cloud.google.com/docs/authentication). After you have authorized access, you can then use the API to start new builds, view build status and details, list builds per project, and cancel builds that are currently in process.

For more information, see the [API documentation](https://cloud.google.com/build/docs/api/reference/rest).

## Default pools and private pools

By default, when you run a build on Cloud Build, the build runs in a secure, hosted environment with access to the public internet. Each build runs on its own **worker** and is isolated from other workloads. You can customize your build in multiple ways including increasing the size of the machine type or allocating more disk space. The default pool has limits on how much you can customize the environment, particularly around private network access.

**Private pools** are private, dedicated pools of workers that offer greater customization over the build environment, including the ability to access resources in a private network. Private pools, similar to default pools, are hosted and fully-managed by Cloud Build and scale up and down to zero, with no infrastructure to set up, upgrade, or scale. Because private pools are customer-specific resources, you can configure them in more ways. 

To learn more about private pools and the feature difference between default pool and private pool, see [Private pool overview](https://cloud.google.com/build/docs/private-pools/private-pools-overview). 

## Build security

Cloud Build provides several features to secure your builds including:

  * **Automated Builds**

An automated build or scripted build defines all build steps in build script or build configuration, including steps to retrieve source code and steps to build the code. The only manual command, if any, is the command to run the build. Cloud Build uses a [build config](https://cloud.google.com/build/docs/build-config) file to provide build steps to Cloud Build.

Automated builds provide consistency in the build steps. However, it's also important to run builds in a consistent, trusted environment.

Although local builds can be useful for debugging purposes, releasing software from local builds can introduce a lot of security concerns, inconsistencies and inefficiencies into the build process.

    * Allowing local builds provides a way for an attacker with malicious intent to modify the build process.
    * Inconsistencies in developer local environments and developer practices make it difficult to reproduce builds and diagnose build issues.

In the [requirements](https://slsa.dev/spec/v0.1/requirements) for the [SLSA](https://slsa.dev) framework, automated builds are a requirement for SLSA level 1, and using a build service instead of developer environments for builds is a requirement for SLSA level 2.

  * **Build provenance**

Build provenance is a collection of verifiable data about a build.

Provenance metadata includes details such as the digests of the built images, the input source locations, the build toolchain, and the build duration.

Generating build provenance helps you to:

    * Verify that a built artifact was created from trusted source location and by a trusted build system.
    * Identify code injected from an untrusted source location or build system.

You can use alerting and policy mechanisms to proactively use build provenance data. For example, you can create policies that only allow deployments of code built from verified sources.

Cloud Build can generate build provenance for container images that provide SLSA level 3 assurance. For more information, see [Viewing build provenance](https://cloud.google.com/build/docs/securing-builds/view-build-provenance).

  * **Ephemeral build environment**

Ephemeral environments are temporary environments that are meant to last for a a single build invocation. After the build, the environment is wiped or deleted. Ephemeral builds ensure that the build service and build steps run in an ephemeral environment, such as a container or VM. Instead of reusing an existing build environment, the build service provisions a new environment for each build and then destroys it after the build process is complete.

Ephemeral environments ensure clean builds since there are no residual files or environment settings from previous builds that can interfere with the build process. A non-ephemeral environment provides an opportunity for attackers to inject malicious files and content. An ephemeral environment also reduces maintenance overhead and reduces inconsistencies in the build environment.

[Cloud Build](https://cloud.google.com/build/docs/overview) sets up a new virtual machine environment for every build and destroys it after the build.

  * **Deployment policies**

You can integrate [Cloud Build with Binary Authorization](https://cloud.google.com/build/docs/securing-builds/secure-deployments-to-run-gke) to check for build attestations and block deployments of images that are not generated by Cloud Build. This process can reduce the risk of deploying unauthorized software.

  * **Customer-managed encryption keys**

Cloud Build provides [customer-managed encryption keys (CMEK)](https://cloud.google.com/kms/docs/cmek#cmek_compliance) compliance by default. Users do not need to configure anything specifically. Cloud Build provides CMEK compliance by encrypting the build-time persistent disk (PD) with an ephemeral key that is generated for each build. The key is uniquely generated for each build.

As soon as the build completes, the key is wiped from memory and destroyed. It is not stored anywhere, is not accessible to Google engineers or support staff, and cannot be restored. The data that was protected using such a key is permanently inaccessible. For more information, see [CMEK compliance in Cloud Build](https://cloud.google.com/build/docs/securing-builds/cmek).

  * **Security insights panel**

Cloud Build includes a **Security insights** panel in the Google Cloud console that displays a high-level overview of multiple security metrics. You can use this panel to identify and mitigate risks in your build process.

This panel displays the following information:

    * **Supply-chain Levels for Software Artifacts (SLSA) Level** : Identifies the maturity level of your software build process in accordance with the [SLSA specification](https://slsa.dev/spec/v0.1/levels).
    * **Vulnerabilities** : An overview of any vulnerabilities found in your artifacts, and the name of the image that [Artifact Analysis](https://cloud.google.com/artifact-analysis/docs/artifact-analysis) has scanned. You can click the image name to view vulnerability details.
    * **Build details** : Details of the build such as the builder and the link to view logs.
    * **Build provenance** : Provenance for the build.



To learn how you can use Cloud Build with other Google Cloud products and features to safeguard your software supply chain, see [Software supply chain security](https://cloud.google.com/software-supply-chain-security/docs/overview).

## What's next

* Read the [Docker quickstart](https://cloud.google.com/build/docs/quickstart-docker) to learn how to use Cloud Build to build Docker images.
* Learn how to [build, test, and deploy artifacts](https://cloud.google.com/build/docs/configuring-builds/build-test-deploy-artifacts) in Cloud Build.
* Learn about different types of [Cloud Build triggers](https://cloud.google.com/build/docs/triggers).
* Read our resources about [DevOps](https://cloud.google.com/devops/) and explore the [DevOps Research and Assessment](https://dora.dev/) research program.

Send feedback 

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2026-02-25 UTC.

Need to tell us more?  [[["Easy to understand","easyToUnderstand","thumb-up"],["Solved my problem","solvedMyProblem","thumb-up"],["Other","otherUp","thumb-up"]],[["Hard to understand","hardToUnderstand","thumb-down"],["Incorrect information or sample code","incorrectInformationOrSampleCode","thumb-down"],["Missing the information/samples I need","missingTheInformationSamplesINeed","thumb-down"],["Other","otherDown","thumb-down"]],["Last updated 2026-02-25 UTC."],[],[]]
