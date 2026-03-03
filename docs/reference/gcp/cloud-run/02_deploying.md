# Deploying container images to Cloud Run  |  Google Cloud Documentation

Source: https://docs.cloud.google.com/run/docs/deploying

* [ Home ](https://docs.cloud.google.com/)
  * [ Documentation ](https://docs.cloud.google.com/docs)
  * [ Application hosting ](https://docs.cloud.google.com/docs/application-hosting)
  * [ Cloud Run ](https://docs.cloud.google.com/run/docs)
  * [ Guides ](https://docs.cloud.google.com/run/docs/overview/what-is-cloud-run)



Send feedback 

#  Deploying container images to Cloud Run Stay organized with collections  Save and categorize content based on your preferences. 

This page describes how to deploy container images to a new Cloud Run service or to a new revision of an existing Cloud Run service.

The container image is imported by Cloud Run when deployed. Cloud Run keeps this copy of the container image as long as it is used by a serving revision. Container images are not pulled from their container repository when a new Cloud Run instance is started.

For an example walkthrough of deploying a new service, see [Deploy a sample container quickstart](https://cloud.google.com/run/docs/quickstarts/deploy-container).

## Before you start

If you are under a domain restriction organization policy [restricting](https://cloud.google.com/resource-manager/docs/organization-policy/restricting-domains) unauthenticated invocations for your project, you will need to access your deployed service as described under [Testing private services](https://cloud.google.com/run/docs/triggering/https-request#testing-private).

### Required roles

To get the permissions that you need to deploy Cloud Run services, ask your administrator to grant you the following IAM roles : 

  * [Cloud Run Developer ](https://cloud.google.com/iam/docs/roles-permissions/run#run.developer) (`roles/run.developer`) on the Cloud Run service
  * [Service Account User ](https://cloud.google.com/iam/docs/roles-permissions/iam#iam.serviceAccountUser) (`roles/iam.serviceAccountUser`) on the service identity
  * [Artifact Registry Reader ](https://cloud.google.com/iam/docs/roles-permissions/artifactregistry#artifactregistry.reader) (`roles/artifactregistry.reader`) on the Artifact Registry repository of the deployed container image
  * If you are using a cross-project service account to deploy a service: [Service Account Token Creator ](https://cloud.google.com/iam/docs/roles-permissions/iam#iam.serviceAccountTokenCreator) (`roles/iam.serviceAccountTokenCreator`) on the service identity 



For a list of IAM roles and permissions that are associated with Cloud Run, see [Cloud Run IAM roles](https://cloud.google.com/run/docs/reference/iam/roles) and [Cloud Run IAM permissions](https://cloud.google.com/run/docs/reference/iam/permissions). If your Cloud Run service interfaces with Google Cloud APIs, such as Cloud Client Libraries, see the [service identity configuration guide](https://cloud.google.com/run/docs/configuring/services/service-identity). For more information about granting roles, see [deployment permissions](https://cloud.google.com/run/docs/reference/iam/roles#additional-configuration) and [manage access](https://cloud.google.com/iam/docs/granting-changing-revoking-access).

## Supported container registries and images

You can directly use container images stored in [Artifact Registry](https://cloud.google.com/artifact-registry/docs/overview), or [Docker Hub](https://hub.docker.com/). Google recommends the use of Artifact Registry. Docker Hub images are [cached](https://cloud.google.com/artifact-registry/docs/pull-cached-dockerhub-images) for up to one hour.

You can use container images from other public or private registries (like JFrog Artifactory, Nexus, or GitHub Container Registry), by setting up an [Artifact Registry remote repository](https://cloud.google.com/artifact-registry/docs/repositories/remote-repo).

You should only consider [Docker Hub](https://hub.docker.com/) for deploying popular container images such as [Docker Official Images](https://docs.docker.com/docker-hub/official_images/) or [Docker Sponsored OSS images](https://docs.docker.com/docker-hub/dsos-program/). For higher availability, Google recommends deploying these Docker Hub images using an [Artifact Registry remote repository](https://cloud.google.com/artifact-registry/docs/repositories/remote-repo).

Cloud Run does not support container image layers larger than 9.9 GB when deploying from Docker Hub or an Artifact Registry remote repository with an external registry.

## Deploying a new service

You can specify a container image with a tag (for example, `us-docker.pkg.dev/my-project/container/my-image:latest`) or with an exact digest (for example, `us-docker.pkg.dev/my-project/container/my-image@sha256:41f34ab970ee...`).

Deploying to a service for the first time creates its first revision. Note that revisions are immutable. If you deploy from a container image tag, it will be resolved to a digest and the revision will always serve this particular digest.

Click the tab for instructions using the tool of your choice.

### Console

To deploy a container image:

  1. In the Google Cloud console, go to the Cloud Run page:

[Go to Cloud Run](https://console.cloud.google.com/run)

  2. Click **Deploy container** to display the **Create service** form.

    1. In the form, select the deployment option:

      1. If you want to manually deploy a container, select **Deploy one revision from an existing container image** and specify the container image.

      2. If you want to automate for continuous deployment, select **Continuously deploy new revisions from a source repository** and follow the [instructions for continuous deployments](https://cloud.google.com/run/docs/continuous-deployment-with-cloud-build#setup-cd).

    2. Enter the needed service name. Service names must be 49 characters or less and must be unique per region and project. A service name cannot be changed later and is publicly visible.

    3. Select the region where you want your service located. The region selector indicates [price tier](https://cloud.google.com/run/pricing), availability of [domain mappings](https://cloud.google.com/run/docs/mapping-custom-domains) and highlights regions with the [lowest carbon impact](https://cloud.google.com/sustainability/region-carbon#region-picker).

    4. Set [billing](https://cloud.google.com/run/docs/configuring/billing-settings) as needed.

    5. Under **Service scaling** , if you use the default Cloud Run [autoscaling](https://cloud.google.com/run/docs/about-instance-autoscaling), optionally specify the [minimum](https://cloud.google.com/run/docs/configuring/min-instances) instances. If you use [manual scaling](https://cloud.google.com/run/docs/configuring/services/manual-scaling), specify the number of instances for the service.

    6. Set the [_Ingress_](https://cloud.google.com/run/docs/securing/ingress) settings in the form as needed.

    7. Under _Authentication_ , configure the following:

       * If you are creating a public API or website, select **Allow public access**. Selecting this assigns the IAM Invoker role to the special identifier `allUser`. You can [use IAM to edit this setting](https://cloud.google.com/run/docs/securing/authenticating#service-to-service) later after you create the service.
       * If you want a secure service protected by authentication, select **Require authentication**.
  3. Click **Container(s), Volumes, Networking, Security** to set other optional settings in the appropriate tabs:

     * [Concurrency](https://cloud.google.com/run/docs/configuring/concurrency)
     * [Container configuration](https://cloud.google.com/run/docs/configuring/services/containers)
     * [CPU limits](https://cloud.google.com/run/docs/configuring/services/cpu)
     * [Memory limits](https://cloud.google.com/run/docs/configuring/services/memory-limits)
     * [Request timeout](https://cloud.google.com/run/docs/configuring/request-timeout)
     * [Secrets](https://cloud.google.com/run/docs/configuring/services/secrets)
     * [Environment variables](https://cloud.google.com/run/docs/configuring/services/environment-variables)
     * [Execution environment](https://cloud.google.com/run/docs/configuring/execution-environments)
     * [HTTP/2](https://cloud.google.com/run/docs/configuring/http2)
     * [Service accounts](https://cloud.google.com/run/docs/configuring/services/service-identity)
     * [Cloud SQL connections](https://cloud.google.com/run/docs/configuring/connect-cloudsql)
     * [VPC connection](https://cloud.google.com/run/docs/configuring/connecting-vpc)
  4. When you are finished configuring your service, click **Create** to deploy the image to Cloud Run and wait for the deployment to finish.

  5. Click the displayed URL link to open the unique and stable endpoint of your deployed service.




###  gcloud 

  1. In the Google Cloud console, activate Cloud Shell.

[Activate Cloud Shell](https://console.cloud.google.com/?cloudshell=true)

At the bottom of the Google Cloud console, a [Cloud Shell](https://cloud.google.com/shell/docs/how-cloud-shell-works) session starts and displays a command-line prompt. Cloud Shell is a shell environment with the Google Cloud CLI already installed and with values already set for your current project. It can take a few seconds for the session to initialize. 

  2. To deploy a container image:

    1. Run the following command: 
[code]        gcloud run deploy SERVICE --image IMAGE_URL
[/code]

Replace the following:

       * SERVICE: the name of the service you want to deploy to. Service names must be 49 characters or less and must be unique per region and project. If the service does not exist yet, this command creates the service during the deployment. You can omit this parameter entirely, but you will be prompted for the service name if you omit it.
       * IMAGE_URL: a reference to the container image, for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`. If you use Artifact Registry, the [repository](https://cloud.google.com/artifact-registry/docs/repositories/create-repos#docker) REPO_NAME must already be created. The URL follows the format of `LOCATION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG` . Note that if you don't supply the `--image` flag, the deploy command will attempt to [deploy from source code](https://cloud.google.com/run/docs/deploying-source-code).

If you are creating a public API or website, allow public access of your service using the `--allow-unauthenticated` flag. This [assigns the **Cloud Run Invoker** IAM role](https://cloud.google.com/run/docs/securing/managing-access#making_a_service_public) to `allUsers`. You can also specify `--no-allow-unauthenticated` to disallow public access. If you omit either of these flags, you are prompted to confirm when the `deploy` command runs.

    2. Wait for the deployment to finish. Upon successful completion, a success message is displayed along with the URL of the deployed service.

_Note that to deploy to a different location_ from the one you set using the `run/region` `gcloud` properties, use:
[code]     gcloud run deploy SERVICE --region REGION
[/code]




### YAML

You can store your service specification in a `YAML` file and then deploy it using the gcloud CLI.

  1. Create a new `service.yaml` file with the following content:
[code]     apiVersion: serving.knative.dev/v1
    kind: Service
    metadata:
      name: SERVICE
    spec:
      template:
        spec:
          containers:
          - image: IMAGE
[/code]

Replace the following:

     * SERVICE: the name of your Cloud Run service. Service names must be 49 characters or less and must be unique per region and project.
     * IMAGE: the URL of your container image.

You can also specify more configuration such as environment variables or

  2. Deploy the new service using the following command:
[code]     gcloud run services replace service.yaml
[/code]

  3. Optionally, [make your service public](https://cloud.google.com/run/docs/authenticating/public) if you want to allow unauthenticated access to the service.




### Terraform

To learn how to apply or remove a Terraform configuration, see [Basic Terraform commands](https://cloud.google.com/docs/terraform/basic-commands).

Add the following to a [`google_cloud_run_v2_service`](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service) resource in your Terraform configuration:  

[code] 
      provider "google" {
        project = "PROJECT-ID"
      }
    
      resource "google_cloud_run_v2_service" "default" {
        name     = "SERVICE"
        location = "REGION"
        client   = "terraform"
    
        template {
          containers {
            image = "IMAGE_URL"
          }
        }
      }
    
      resource "google_cloud_run_v2_service_iam_member" "noauth" {
        location = google_cloud_run_v2_service.default.location
        name     = google_cloud_run_v2_service.default.name
        role     = "roles/run.invoker"
        member   = "allUsers"
      }
    
[/code]

Replace the following:

  * PROJECT-ID: the Google Cloud project ID
  * REGION: the Google Cloud region
  * SERVICE: the name of your Cloud Run service. Service names must be 49 characters or less and must be unique per region and project.
  * IMAGE_URL: a reference to the container image, for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`. If you use Artifact Registry, the [repository](https://cloud.google.com/artifact-registry/docs/repositories/create-repos#docker) REPO_NAME must already be created. The URL follows the format of `LOCATION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG`



This configuration allows public access (the equivalent of `--allow-unauthenticated`). To make the service private, remove the `google_cloud_run_v2_service_iam_member` stanza.

###  Compose 

**Preview — Cloud Run Compose deploy**

This feature is subject to the "Pre-GA Offerings Terms" in the General Service Terms section of the [Service Specific Terms](https://cloud.google.com/terms/service-terms#1). Pre-GA features are available "as is" and might have limited support. For more information, see the [launch stage descriptions](https://cloud.google.com/products/#product-launch-stages). 

You can store your [Compose Specification](https://compose-spec.io) in a `YAML` file and then deploy it as a Cloud Run service using a [single gcloud command](https://cloud.google.com/run/docs/deploy-run-compose).

To deploy a `compose.yaml` file as a Cloud Run service, follow these steps:

  1. In your project directory, create a `compose.yaml` file with your service definitions.
[code]     services:
      web:
        image: IMAGE
        ports:
          - "8080:8080"
[/code]

Replace IMAGE with the URL of your container image.

You can also specify more configuration options such as environment variables, secrets, and volume mounts.

  2. To deploy the services, run the `gcloud beta run compose up` command:
[code]     gcloud beta run compose up compose.yaml
[/code]

  3. Respond `y` to any prompts to install required components or to enable APIs.

  4. Optional: [Make your service public](https://cloud.google.com/run/docs/authenticating/public) if you want to allow unauthenticated access to the service.




After deployment, the Cloud Run service URL is displayed. Copy this URL and paste it into your browser to view the running container. You can disable the default authentication from the Google Cloud console.

### Client libraries

To deploy a new service from code:

  * [Go](https://cloud.google.com/go/docs/reference/cloud.google.com/go/run/latest/apiv2#cloud_google_com_go_run_apiv2_ServicesClient_CreateService)
  * [Java](https://cloud.google.com/java/docs/reference/google-cloud-run/latest/com.google.cloud.run.v2.ServicesClient#com_google_cloud_run_v2_ServicesClient_createServiceAsync_com_google_cloud_run_v2_CreateServiceRequest_)
  * [Node.js](https://cloud.google.com/nodejs/docs/reference/run/latest/run/v2.servicesclient#_google_cloud_run_v2_ServicesClient_createService_member_1_)
  * [Python](https://cloud.google.com/python/docs/reference/run/latest/google.cloud.run_v2.services.services.ServicesClient#google_cloud_run_v2_services_services_ServicesClient_create_service)
  * [Ruby](https://cloud.google.com/ruby/docs/reference/google-cloud-run-v2/latest/Google-Cloud-Run-V2-Services-Client#Google__Cloud__Run__V2__Services__Client_create_service_instance_)
  * [PHP](https://cloud.google.com/php/docs/reference/cloud-run/latest/V2.Client.ServicesClient#_Google_Cloud_Run_V2_Client_ServicesClient__createService__)
  * [.NET](https://cloud.google.com/dotnet/docs/reference/Google.Cloud.Run.V2/latest/Google.Cloud.Run.V2.ServicesClient#Google_Cloud_Run_V2_ServicesClient_CreateService_Google_Api_Gax_ResourceNames_LocationName_Google_Cloud_Run_V2_Service_System_String_Google_Api_Gax_Grpc_CallSettings_)



### REST API

To deploy a new service, send a `POST` HTTP request to the Cloud Run Admin API [`service` endpoint](https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/create).

For example, using `curl`:
[code] 
    curl -H "Content-Type: application/json" \
      -H "Authorization: Bearer ACCESS_TOKEN" \
      -X POST \
      -d '{template: {containers: [{image: "IMAGE_URL"}]}}' \
      https://run.googleapis.com/v2/projects/PROJECT_ID/locations/REGION/services?serviceId=SERVICE
[/code]

Replace the following:

  * ACCESS_TOKEN: a valid access token for an account that has the [IAM permissions to deploy services](https://cloud.google.com/run/docs/reference/iam/permissions). For example, if you are logged into gcloud, you can retrieve an access token using `gcloud auth print-access-token`. From within a Cloud Run container instance, you can retrieve an access token using the [container instance metadata server](https://cloud.google.com/run/docs/container-contract#metadata-server).
  * IMAGE_URL: a reference to the container image, for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`. If you use Artifact Registry, the [repository](https://cloud.google.com/artifact-registry/docs/repositories/create-repos#docker) REPO_NAME must already be created. The URL follows the format of `LOCATION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG` .
  * SERVICE: the name of the service you want to deploy to. Service names must be 49 characters or less and must be unique per region and project.
  * REGION: the Google Cloud region of the service.
  * PROJECT-ID: the Google Cloud project ID.



### Cloud Run locations

Cloud Run is regional, which means the infrastructure that runs your Cloud Run services is located in a specific region and is managed by Google to be redundantly available across [all the zones within that region](https://cloud.google.com/docs/geography-and-regions).   
  


Meeting your latency, availability, or durability requirements are primary factors for selecting the region where your Cloud Run services are run. You can generally select the region nearest to your users but you should consider the location of the [other Google Cloud products](https://cloud.google.com/about/locations/#locations) that are used by your Cloud Run service. Using Google Cloud products together across multiple locations can affect your service's latency as well as cost.  
  


Cloud Run is available in the following regions:

#### Subject to [Tier 1 pricing](https://cloud.google.com/run/pricing#tables)

  * `asia-east1` (Taiwan) 
  * `asia-northeast1` (Tokyo) 
  * `asia-northeast2` (Osaka) 
  * `asia-south1` (Mumbai, India) 
  * `asia-southeast3 ` (Bangkok) 
  * `europe-north1` (Finland) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-north2` (Stockholm) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-southwest1` (Madrid) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west1` (Belgium) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west4` (Netherlands) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west8` (Milan) 
  * `europe-west9` (Paris) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `me-west1` (Tel Aviv) 
  * `northamerica-south1` (Mexico) 
  * `us-central1` (Iowa) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `us-east1` (South Carolina) 
  * `us-east4` (Northern Virginia) 
  * `us-east5` (Columbus) 
  * `us-south1` (Dallas) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `us-west1` (Oregon) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)



#### Subject to [Tier 2 pricing](https://cloud.google.com/run/pricing#tables)

  * `africa-south1` (Johannesburg) 
  * `asia-east2` (Hong Kong) 
  * `asia-northeast3` (Seoul, South Korea) 
  * `asia-southeast1` (Singapore) 
  * `asia-southeast2 ` (Jakarta) 
  * `asia-south2` (Delhi, India) 
  * `australia-southeast1` (Sydney) 
  * `australia-southeast2` (Melbourne) 
  * `europe-central2` (Warsaw, Poland) 
  * `europe-west10` (Berlin) 
  * `europe-west12` (Turin) 
  * `europe-west2` (London, UK) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west3` (Frankfurt, Germany) 
  * `europe-west6` (Zurich, Switzerland) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `me-central1` (Doha) 
  * `me-central2` (Dammam) 
  * `northamerica-northeast1` (Montreal) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `northamerica-northeast2` (Toronto) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `southamerica-east1` (Sao Paulo, Brazil) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `southamerica-west1` (Santiago, Chile) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `us-west2` (Los Angeles) 
  * `us-west3` (Salt Lake City) 
  * `us-west4` (Las Vegas) 



If you already created a Cloud Run service, you can view the region in the Cloud Run dashboard in the [Google Cloud console](https://console.cloud.google.com/run).

OK

## Deploying a new revision of an existing service

You can deploy a new revision using the Google Cloud console, the `gcloud` command line, or a YAML configuration file.

Note that changing any configuration settings results in the creation of a new revision, even if there is no change to the container image. Each revision created is immutable.

The container image is imported by Cloud Run when deployed. Cloud Run keeps this copy of the container image as long as it is used by a serving revision.

Click the tab for instructions using the tool of your choice.

### Console

To deploy a new revision of an existing service:

  1. In the Google Cloud console, go to the Cloud Run **Services** page:

[Go to Cloud Run](https://console.cloud.google.com/run/services)

  2. Locate the service you want to update in the services list, and click to open the details of that service.

  3. Click **Edit and deploy new revision** to display the revision deployment form.

    1. If needed, supply the URL to the new container image you want to deploy.

    2. [Configure the container](https://cloud.google.com/run/docs/configuring/services/containers) as needed.

    3. Set [billing](https://cloud.google.com/run/docs/configuring/billing-settings) as needed.

    4. Under Capacity, specify [memory limits](https://cloud.google.com/run/docs/configuring/services/memory-limits). and [CPU limits](https://cloud.google.com/run/docs/configuring/services/cpu).

    5. Specify [request timeout](https://cloud.google.com/run/docs/configuring/request-timeout) and [concurrency](https://cloud.google.com/run/docs/configuring/concurrency) as needed.

    6. Specify [execution environment](https://cloud.google.com/run/docs/configuring/execution-environments) as needed.

    7. Under _Autoscaling_ , specify [minimum](https://cloud.google.com/run/docs/configuring/min-instances) and [maximum](https://cloud.google.com/run/docs/configuring/max-instances) instances.

    8. Use the other tabs as needed to optionally configure:

       * [Secrets](https://cloud.google.com/run/docs/configuring/services/secrets)
       * [Environment variables](https://cloud.google.com/run/docs/configuring/services/environment-variables)
       * [HTTP/2](https://cloud.google.com/run/docs/configuring/http2)
       * [Service accounts](https://cloud.google.com/run/docs/configuring/services/service-identity)
       * [Cloud SQL connections](https://cloud.google.com/run/docs/configuring/connect-cloudsql)
       * [VPC connection](https://cloud.google.com/run/docs/configuring/connecting-vpc)
  4. To send all traffic to the new revision, select **Serve this revision immediately**. To gradually roll out a new revision, clear that checkbox. This results in a deployment where no traffic is sent to the new revision. Follow the instructions for [gradual rollouts](https://cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration#gradual) after you deploy.

  5. Click **Deploy** and wait for the deployment to finish.




###  gcloud 

  1. In the Google Cloud console, activate Cloud Shell.

[Activate Cloud Shell](https://console.cloud.google.com/?cloudshell=true)

At the bottom of the Google Cloud console, a [Cloud Shell](https://cloud.google.com/shell/docs/how-cloud-shell-works) session starts and displays a command-line prompt. Cloud Shell is a shell environment with the Google Cloud CLI already installed and with values already set for your current project. It can take a few seconds for the session to initialize. 

  2. To deploy a container image:

    1. Run the command: 
[code]        gcloud run deploy SERVICE --image IMAGE_URL
[/code]

Replace the following:

       * SERVICE: the name of the service you are deploying to. You can omit this parameter entirely, but you will be prompted for the service name if you omit it.
       * IMAGE_URL: a reference to the container image, for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`. If you use Artifact Registry, the [repository](https://cloud.google.com/artifact-registry/docs/repositories/create-repos#docker) REPO_NAME must already be created. The URL follows the format of `LOCATION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG` .

The revision suffix is assigned automatically for new revisions. If you want to supply your own revision suffix, use the gcloud CLI parameter [\--revision-suffix](https://cloud.google.com/sdk/gcloud/reference/run/deploy#--revision-suffix).

    2. Wait for the deployment to finish. Upon successful completion, a success message is displayed along with the URL of the deployed service.




### YAML

**Caution:** The following instructions replaces your existing service configuration with the one specified in the YAML file. So if you use YAML to make revision changes, you should avoid also using the Google Cloud console or gcloud CLI to make configuration changes because those can be overwritten when you use YAML.

If you need to download or view the configuration of an existing service, use the following command to save results to a YAML file:
[code] 
    gcloud run services describe SERVICE --format export > service.yaml
[/code]

From a service configuration YAML file, modify any `spec.template` child attributes as needed to update revision settings, then deploy the new revision:
[code] 
    gcloud run services replace service.yaml
[/code]

### Cloud Code

To deploy a new revision of an existing service with [Cloud Code](https://cloud.google.com/code/docs), read the [IntelliJ](https://cloud.google.com/code/docs/intellij/deploying-a-cloud-run-app) and [Visual Studio Code](https://cloud.google.com/code/docs/vscode/deploying-a-cloud-run-app) guides.

### Terraform

Make sure you have setup Terraform as described in the Deploying a new service example.

  1. Make a change to the configuration file.

  2. Apply the Terraform configuration:
[code]     terraform apply
[/code]

Confirm you want to apply the actions described by entering `yes`.

**Note:** Unless a configuration change is required, no new revision is created.



###  Compose 

**Preview — Cloud Run Compose deploy**

This feature is subject to the "Pre-GA Offerings Terms" in the General Service Terms section of the [Service Specific Terms](https://cloud.google.com/terms/service-terms#1). Pre-GA features are available "as is" and might have limited support. For more information, see the [launch stage descriptions](https://cloud.google.com/products/#product-launch-stages). 

You can store your [Compose Specification](https://compose-spec.io) in a `YAML` file and then deploy it as a Cloud Run service revision [using a single gcloud command](https://cloud.google.com/run/docs/deploy-run-compose).

To deploy a `compose.yaml` file as a Cloud Run service revision, follow these steps:

  1. In your project directory, create a `compose.yaml` file with your service definitions.
[code]     services:
      web:
        image: IMAGE
        ports:
          - "8080:8080"
[/code]

Replace IMAGE with the URL of your container image.

You can also specify more configuration options such as environment variables, secrets, and volume mounts.

  2. To deploy the services, run the `gcloud beta run compose up` command:
[code]     gcloud beta run compose up compose.yaml
[/code]

  3. Respond `y` to any prompts to install required components or to enable APIs.

  4. Optional: [Make your service public](https://cloud.google.com/run/docs/authenticating/public) if you want to allow unauthenticated access to the service.




After deployment, the Cloud Run service URL is displayed. Copy this URL and paste it into your browser to view the running container. You can disable the default authentication from the Google Cloud console.

### Client libraries

To deploy a new revision from code:

  * [Go](https://cloud.google.com/go/docs/reference/cloud.google.com/go/run/latest/apiv2#cloud_google_com_go_run_apiv2_ServicesClient_UpdateService)
  * [Java](https://cloud.google.com/java/docs/reference/google-cloud-run/latest/com.google.cloud.run.v2.ServicesClient#com_google_cloud_run_v2_ServicesClient_updateServiceAsync_com_google_cloud_run_v2_Service_)
  * [Node.js](https://cloud.google.com/nodejs/docs/reference/run/latest/run/v2.servicesclient#_google_cloud_run_v2_ServicesClient_updateService_member_1_)
  * [Python](https://cloud.google.com/python/docs/reference/run/latest/google.cloud.run_v2.services.services.ServicesClient#google_cloud_run_v2_services_services_ServicesClient_update_service)
  * [Ruby](https://cloud.google.com/ruby/docs/reference/google-cloud-run-v2/latest/Google-Cloud-Run-V2-Services-Client#Google__Cloud__Run__V2__Services__Client_update_service_instance_)
  * [PHP](https://cloud.google.com/php/docs/reference/cloud-run/latest/V2.Client.ServicesClient#_Google_Cloud_Run_V2_Client_ServicesClient__updateService__)
  * [.NET](https://cloud.google.com/dotnet/docs/reference/Google.Cloud.Run.V2/latest/Google.Cloud.Run.V2.ServicesClient#Google_Cloud_Run_V2_ServicesClient_UpdateService_Google_Cloud_Run_V2_Service_Google_Api_Gax_Grpc_CallSettings_)



### REST API

To deploy a new revision, send a `PATCH` HTTP request to the Cloud Run Admin API [`service` endpoint](https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/patch).

For example, using `curl`:
[code] 
    curl -H "Content-Type: application/json" \
      -H "Authorization: Bearer ACCESS_TOKEN" \
      -X PATCH \
      -d '{template: {containers: [{image: "IMAGE_URL"}]}}' \
      https://run.googleapis.com/v2/projects/PROJECT_ID/locations/REGION/services/SERVICE
[/code]

Replace the following:

  * ACCESS_TOKEN: a valid access token for an account that has the [IAM permissions to deploy revisions](https://cloud.google.com/run/docs/reference/iam/permissions). For example, if you are logged into gcloud, you can retrieve an access token using `gcloud auth print-access-token`. From within a Cloud Run container instance, you can retrieve an access token using the [container instance metadata server](https://cloud.google.com/run/docs/container-contract#metadata-server).
  * IMAGE_URL: a reference to the container image, for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`. If you use Artifact Registry, the [repository](https://cloud.google.com/artifact-registry/docs/repositories/create-repos#docker) REPO_NAME must already be created. The URL follows the format of `LOCATION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG` .
  * SERVICE: the name of the service you are deploying to.
  * REGION: the Google Cloud region of the service.
  * PROJECT-ID: the Google Cloud project ID.



### Cloud Run locations

Cloud Run is regional, which means the infrastructure that runs your Cloud Run services is located in a specific region and is managed by Google to be redundantly available across [all the zones within that region](https://cloud.google.com/docs/geography-and-regions).   
  


Meeting your latency, availability, or durability requirements are primary factors for selecting the region where your Cloud Run services are run. You can generally select the region nearest to your users but you should consider the location of the [other Google Cloud products](https://cloud.google.com/about/locations/#locations) that are used by your Cloud Run service. Using Google Cloud products together across multiple locations can affect your service's latency as well as cost.  
  


Cloud Run is available in the following regions:

#### Subject to [Tier 1 pricing](https://cloud.google.com/run/pricing#tables)

  * `asia-east1` (Taiwan) 
  * `asia-northeast1` (Tokyo) 
  * `asia-northeast2` (Osaka) 
  * `asia-south1` (Mumbai, India) 
  * `asia-southeast3 ` (Bangkok) 
  * `europe-north1` (Finland) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-north2` (Stockholm) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-southwest1` (Madrid) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west1` (Belgium) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west4` (Netherlands) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west8` (Milan) 
  * `europe-west9` (Paris) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `me-west1` (Tel Aviv) 
  * `northamerica-south1` (Mexico) 
  * `us-central1` (Iowa) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `us-east1` (South Carolina) 
  * `us-east4` (Northern Virginia) 
  * `us-east5` (Columbus) 
  * `us-south1` (Dallas) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `us-west1` (Oregon) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)



#### Subject to [Tier 2 pricing](https://cloud.google.com/run/pricing#tables)

  * `africa-south1` (Johannesburg) 
  * `asia-east2` (Hong Kong) 
  * `asia-northeast3` (Seoul, South Korea) 
  * `asia-southeast1` (Singapore) 
  * `asia-southeast2 ` (Jakarta) 
  * `asia-south2` (Delhi, India) 
  * `australia-southeast1` (Sydney) 
  * `australia-southeast2` (Melbourne) 
  * `europe-central2` (Warsaw, Poland) 
  * `europe-west10` (Berlin) 
  * `europe-west12` (Turin) 
  * `europe-west2` (London, UK) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `europe-west3` (Frankfurt, Germany) 
  * `europe-west6` (Zurich, Switzerland) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `me-central1` (Doha) 
  * `me-central2` (Dammam) 
  * `northamerica-northeast1` (Montreal) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `northamerica-northeast2` (Toronto) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `southamerica-east1` (Sao Paulo, Brazil) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `southamerica-west1` (Santiago, Chile) ![leaf icon](https://cloud.google.com/sustainability/region-carbon/gleaf.svg) [Low CO2](https://cloud.google.com/sustainability/region-carbon#region-picker)
  * `us-west2` (Los Angeles) 
  * `us-west3` (Salt Lake City) 
  * `us-west4` (Las Vegas) 



If you already created a Cloud Run service, you can view the region in the Cloud Run dashboard in the [Google Cloud console](https://console.cloud.google.com/run).

OK

## Deploying images from other Google Cloud projects

To deploy images from other Google Cloud projects, you or your administrator must grant the deployer account and the Cloud Run service agent the required IAM roles.

For the required roles for the deployer account, see required roles.

To grant the Cloud Run service agent the required roles, see the following instructions:

  1. In the Google Cloud console, open the project for your Cloud Run service.

[Go to the IAM page](https://console.cloud.google.com/iam-admin/iam)

  2. Select **Include Google-provided role grants**.

  3. Copy the email of the Cloud Run [service agent](https://cloud.google.com/iam/docs/service-agents). It has the suffix **@serverless-robot-prod.iam.gserviceaccount.com**

  4. Open the project that owns the container registry you want to use.

[Go to the IAM page](https://console.cloud.google.com/iam-admin/iam)

  5. Click **Add** to add a new principal.

  6. In the **New principals** field, paste in the email of the service account that you copied earlier.

  7. In the _Select a role_ menu, click **Artifact Registry - > Artifact Registry Reader**.

  8. Deploy the container image to the project that contains your Cloud Run service.

**Note:** For stronger security, [grant access to the Artifact Registry repository that contains your container images](https://cloud.google.com/artifact-registry/docs/access-control#gcp).



## Deploying images from other registries

To deploy public or private container images that are not stored in Artifact Registry or Docker Hub, [set up](https://cloud.google.com/artifact-registry/docs/repositories/remote-repo) an Artifact Registry [remote repository](https://cloud.google.com/artifact-registry/docs/repositories/remote-overview).

Artifact Registry remote repositories allow you to:

  * Deploy any public container image—for example, GitHub Container Registry (`ghcr.io`).
  * Deploy container images from private repositories that require authentication—for example, JFrog Artifactory or Nexus.



Alternatively, if using an Artifact Registry remote repository is not an option, you can temporarily pull and push container images to [Artifact Registry](https://cloud.google.com/artifact-registry/docs/overview) using `docker push` in order to deploy them to Cloud Run. The container image is imported by Cloud Run when deployed, so after the deployment, you can [delete the image from Artifact Registry](https://cloud.google.com/artifact-registry/docs/docker/manage-images#deleting_images).

## Deploying multiple containers to a service (sidecars)

In a Cloud Run deployment with sidecars, there is one _ingress_ container that handles all incoming HTTPS requests at the container PORT you specify, and there are one or more _sidecar_ containers. The sidecars cannot listen for the incoming HTTP requests at the ingress container port, but they can communicate with each other and with the ingress container using a localhost port. The localhost port used varies depending on the containers you are using.

In the following diagram, the ingress container is communicating with the sidecar using `localhost:5000`.

![Cloud Run multi-container](https://cloud.google.com/static/run/docs/images/multicontainer.png)

You can deploy up to 10 containers per instance including the ingress container. All containers within an instance share the same network namespace and can also share files using an in-memory shared volume, as shown in the diagram.

You can deploy multiple containers in either the first or second generation [execution environment](https://cloud.google.com/run/docs/configuring/execution-environments).

If you use [request-based billing](https://cloud.google.com/run/docs/configuring/billing-settings) (the Cloud Run default), sidecars are allocated CPU in only these scenarios:

  * The instance is processing at least one request.
  * The ingress container is starting up.



If your sidecar must use CPU outside of request processing (for example, for metrics collection), configure your billing setting to instance-based billing for your service. For more information see [Billing settings (services)](https://cloud.google.com/run/docs/configuring/billing-settings).

If you use request-based billing, configure a [startup probe](https://cloud.google.com/run/docs/configuring/healthchecks) to ensure your sidecar is not CPU throttled on startup.

You can require all deployments to use a specific sidecar by creating [custom organization policies](https://cloud.google.com/run/docs/securing/custom-constraints#require-prefix).

### Use cases

Use cases for sidecars in a Cloud Run service include:

  * Application monitoring, logging and tracing
  * Using [Nginx](https://cloud.google.com/run/docs/internet-proxy-nginx-sidecar), Envoy or Apache2 as a proxy in front of your application container
  * Adding authentication and authorization filters (for example, Open Policy Agent)
  * Running outbound connection proxies such as the Alloy DB Auth proxy



### Deploying a service with sidecar containers

You can deploy multiple sidecars to a Cloud Run service using the Google Cloud console, the Google Cloud CLI, YAML, or Terraform.

Click the tab for instructions using the tool of your choice.

### Console

  1. In the Google Cloud console, go to the Cloud Run **Services** page:

[Go to Cloud Run](https://console.cloud.google.com/run/services)

     * To deploy to an existing service, locate it in the services list, and click to open, then click **Edit and deploy a new revision** to display the revision deployment form.
     * To deploy a new service, click **Deploy container** to display the **Create service** form.
  2. For a new service,

    1. Supply the service name and the URL to the ingress container image you want to deploy.
    2. Click **Container(s), Volumes, Networking, Security**
  3. In the **Edit container** card, configure the ingress container as needed.

  4. Click **Add container** and configure a sidecar container you want to add alongside the ingress container. If the sidecar depends on another container in the service, indicate this in the **[Container start-up order](https://cloud.google.com/run/docs/configuring/services/containers#container-ordering)** menu. Repeat this step for each sidecar container you are deploying.

  5. To send all traffic to the new revision, select **Serve this revision immediately**. For a gradual rollout, clear that checkbox. This results in a deployment where no traffic is sent to the new revision. Follow the instructions for [gradual rollouts](https://cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration#gradual) after you deploy.

  6. Click **Create** for a new service or **Deploy** for an existing service, then wait for the deployment to finish.




###  gcloud 

  1. In the Google Cloud console, activate Cloud Shell.

[Activate Cloud Shell](https://console.cloud.google.com/?cloudshell=true)

At the bottom of the Google Cloud console, a [Cloud Shell](https://cloud.google.com/shell/docs/how-cloud-shell-works) session starts and displays a command-line prompt. Cloud Shell is a shell environment with the Google Cloud CLI already installed and with values already set for your current project. It can take a few seconds for the session to initialize. 

  2. To deploy multiple containers to a service, run the following command:
[code]     gcloud run deploy SERVICE \
     --container INGRESS_CONTAINER_NAME \
     --image='INGRESS_IMAGE' \
     --port='CONTAINER_PORT' \
     --container SIDECAR_CONTAINER_NAME \
     --image='SIDECAR_IMAGE'
[/code]

Replace the following:

     * SERVICE: the name of the service you are deploying to. You can omit this parameter entirely, but you will be prompted for the service name if you omit it.
     * INGRESS_CONTAINER_NAME: a name for the container receiving requests—for example `app`.
     * INGRESS_IMAGE: a reference to the container image that should receive requests—for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`.
     * CONTAINER_PORT: the port where the ingress container listens for incoming requests. Unlike a single-container service, for a service containing sidecars, there is no default port for the ingress container. You must explicitly configure the container port for the ingress container and only one container can have the port exposed.
     * SIDECAR_CONTAINER_NAME: a name for the sidecar container—for example `sidecar`.
     * SIDECAR_IMAGE: a reference to the sidecar container image

If you want to configure each container in the deploy command, supply each container's configuration after the `container` parameters, for example:
[code]     gcloud run deploy SERVICE \
      --container CONTAINER_1_NAME \
      --image='INGRESS_IMAGE' \
      --set-env-vars=KEY=VALUE \
      --port='CONTAINER_PORT' \
      --container SIDECAR_CONTAINER_NAME \
      --image='SIDECAR_IMAGE' \
      --set-env-vars=KEY_N=VALUE_N
[/code]

**Important:** When you use the `--container` flag, you must specify all non-container-level flags before the container-level flags, otherwise the deploy command fails with an error message to that effect. For example, in the command "`gcloud run services deploy service --execution-environment=gen2 --container app --memory=1G`", the `--execution-environment` flag must be passed before `--container` flag.
  3. Wait for the deployment to finish. Upon successful completion, a success message is displayed along with the URL of the deployed service.




### YAML

These instructions show a basic YAML file for your Cloud Run service with sidecars. Create a file named `service.yaml` and add the following to it:
[code] 
    apiVersion: serving.knative.dev/v1
    kind: Service
    metadata:
      annotations:
      name: SERVICE
    spec:
      template:
        spec:
          containers:
          - image: INGRESS_IMAGE
            ports:
              - containerPort: CONTAINER_PORT
          - image: SIDECAR_IMAGE
          
[/code]

Replace the following:

  * SERVICE: the name of your Cloud Run service. Service names must be 49 characters or less.
  * CONTAINER_PORT: the port where the ingress container listens for incoming requests. Unlike a single-container service, for a service containing sidecars, there is no default port for the ingress container. You must explicitly configure the container port for the ingress container and only one container can have the port exposed.
  * INGRESS_IMAGE: a reference to the container image that should receive requests—for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`.
  * SIDECAR_IMAGE: a reference to the sidecar container image. You can specify multiple sidecars by adding more elements to the `containers` array in the YAML.



After you update the YAML to include the ingress and sidecar containers, deploy to Cloud Run using the command:
[code] 
    gcloud run services replace service.yaml
[/code]

### Terraform

To learn how to apply or remove a Terraform configuration, see [Basic Terraform commands](https://cloud.google.com/docs/terraform/basic-commands).

Add the following to a [`google_cloud_run_v2_service`](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service) resource in your Terraform configuration:  

[code] 
    resource "google_cloud_run_v2_service" "default" {
      name     = "SERVICE"
      location = "REGION"
      ingress = "INGRESS_TRAFFIC_ALL"
      template {
        containers {
          name = "INGRESS_CONTAINER_NAME"
          ports {
            container_port = CONTAINER_PORT
          }
          image = "INGRESS_IMAGE"
          depends_on = ["SIDECAR_CONTAINER_NAME"]
        }
        containers {
          name = "SIDECAR_CONTAINER_NAME"
          image = "SIDECAR_IMAGE"
          }
        }
      }
    
[/code]

The CONTAINER_PORT represents the port where the ingress container listens for incoming requests. Unlike a single-container service, for a service containing sidecars, there is no default port for the ingress container. You must explicitly configure the container port for the ingress container and only one container can have the port exposed.

### Notable features available to deployments with sidecars

#### Start up order

You can specify the [container start up order](https://cloud.google.com/run/docs/configuring/services/containers#container-ordering) within a deployment with multiple containers, if you have dependencies that require some containers to start up before other containers in the deployment.

If you have containers that depend on other containers, you must use [healthchecks](https://cloud.google.com/run/docs/configuring/healthchecks) in your deployment. If you use healthchecks, Cloud Run follows the container startup order and inspects the health of each container, making sure each passes successfully before Cloud Run starts up the next container in the order. If you don't use healthchecks, healthy containers will start up even if the containers they depend on are not running.

#### Exchanging file data between sidecars

Multiple containers within a single instance can access a shared [in-memory volume](https://cloud.google.com/run/docs/configuring/in-memory-volumes), which is accessible to each container using mount points that you create. This is commonly used to share files between containers, for example a telemetry sidecar container can collect logs from an application container.

#### Communicating between sidecars

Two containers of the same instance can communicate with each other on the local network.

Consider this example service:
[code] 
    apiVersion: serving.knative.dev/v1
    kind: Service
    metadata:
      name: example
    spec:
      template:
        spec:
          containers:
          - name: ingress
            image: ...
            ports:
              - containerPort: 8080
          - name: sidecar
            image: ...
    
[/code]

Each instance of this service will run two containers: one named `ingress` and another named `sidecar`.

Requests reaching the service are sent to the `ingress` container on port `8080`. In a service with multiple containers, only one container can be configured as the ingress container that handles all incoming requests, and this must be the container for which a `containerPort` is configured.

Containers `ingress` and `sidecar` can communicate with each other on `http://localhost`. For example, if the container `sidecar` listens for requests on port `5000`, then the container `ingress` can communicate with it on `http://localhost:5000`.

Because the containers are named, the containers can even communicate with each other using the name of the container. For example, if the container `sidecar` listens for requests on port `5000`, then the container `ingress` can communicate with `sidecar` using on `http://sidecar:5000`.

## Adapt your containers for Cloud Run

Most containers you will build or find will be compatible with the Cloud Run [container runtime contract](https://cloud.google.com/run/docs/container-contract). However, you might need to alter some containers built to ease local development or expect full machine control to make them compatible with Cloud Run execution environments.

### Move mounts to your Cloud Run configuration

Your container's initialization scripts must assume that mounts are already complete before calling your container. You must move any mount operations into your [Cloud Run resource configuration](https://cloud.google.com/run/docs/configuring/services/nfs-volume-mounts).

### Switch to a non-root user when possible

Prefer containers that don't use or rely on the root user. This practice reduces your Cloud Run service's vulnerability risk, decreases the container's attack surface, limits attacker access to your file systems, and adheres to the principle of least privilege.

Use the `USER` instruction in your Dockerfile to switch to a less privileged identity, as the default is to run as root. Cloud Run uses the user specified in your Dockerfile to run your container.

### Audit for use of `setuid` binaries

Execution of `setuid` binaries will fail when run from your containers in Cloud Run.

If you're using Docker or Podman locally, use the `--cap-drop=setuid` argument. Alternatively, validate that the binaries you depend on don't have the `setuid` bit set.

### Verify that root containers are compatible with user namespaces

Test your changes locally or in a VM by evaluating your code when running under user namespaces, such as when using [Docker's `userns-remap`](https://docs.docker.com/engine/security/userns-remap/) feature, running your container in [rootless Podman](https://www.redhat.com/en/blog/rootless-containers-podman), or deploying those changes to VMs running the Container-Optimized OS from Google with the `--userns-remap=default` argument in the `docker run` command.

## Disabling the deployment health check

By default, Cloud Run checks that your deployment is healthy by starting an instance and waiting for its startup probe to pass. If the health check fails, the revision will be marked as unhealthy and the traffic won't be routed to it.

If it is not needed or to increase deployment speed, the deployment health check can be disabled:

###  gcloud 

To disable the deployment health check, use the `--no-deploy-health-check` flag:
[code] 
    gcloud run deploy --image IMAGE_URL --no-deploy-health-check
[/code]

Replace the following:

  * IMAGE_URL: a reference to the container image, for example, `us-docker.pkg.dev/cloudrun/container/hello:latest`. If you use Artifact Registry, the [repository](https://cloud.google.com/artifact-registry/docs/repositories/create-repos#docker) REPO_NAME must already be created. The URL follows the format of `LOCATION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG` .



Use `--deploy-health-check` to re-enable the deployment health check if it was previous disabled.

### YAML

To disable the deployment health check, add the `run.googleapis.com/health-check-disabled` annotation with value `'true'` to `spec.template.metadata.annotations`.
[code] 
    apiVersion: serving.knative.dev/v1
    kind: Service
    metadata:
      name: SERVICE
    spec:
      template:
        metadata:
          annotations:
            run.googleapis.com/health-check-disabled: 'true'
    
[/code]

### Terraform

To disable the deployment health check, set the `health_check_disabled` argument to `true` in the `template` block.
[code] 
    resource "google_cloud_run_v2_service" "default" {
      name     = "SERVICE"
      ...
      template {
        health_check_disabled = true
        ...
      }
    }
    
[/code]

## What's next

After you deploy a new service, you can do the following:

  * [Gradual rollouts, rollback revisions, traffic migration](https://cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration)
  * [View service logs](https://cloud.google.com/run/docs/logging)
  * [Monitor service performances](https://cloud.google.com/run/docs/monitoring)
  * [Set memory limits](https://cloud.google.com/run/docs/configuring/services/memory-limits)
  * [Set environment variables](https://cloud.google.com/run/docs/configuring/services/environment-variables)
  * [Change service concurrency](https://cloud.google.com/run/docs/configuring/concurrency)
  * [Manage the service](https://cloud.google.com/run/docs/managing/services)
  * [Manage service revisions](https://cloud.google.com/run/docs/managing/revisions)
  * [Cloud Run OpenTelemetry sidecar example](https://github.com/GoogleCloudPlatform/opentelemetry-cloud-run)
  * [Deploy only trusted images with Binary Authorization](https://cloud.google.com/binary-authorization/docs/run/enabling-binauthz-cloud-run) ([Preview](https://cloud.google.com/products/#product-launch-stages))



You can automate the builds and deployments of your Cloud Run services using Cloud Build Triggers:

  * [Set up Continuous Deployment](https://cloud.google.com/run/docs/continuous-deployment)



You can also use Cloud Deploy to set up a continuous-delivery pipeline to deploy Cloud Run services to multiple environments:

  * [Deploy an app to Cloud Run using Cloud Deploy](https://cloud.google.com/deploy/docs/deploy-app-run)



Send feedback 

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2026-02-25 UTC.

Need to tell us more?  [[["Easy to understand","easyToUnderstand","thumb-up"],["Solved my problem","solvedMyProblem","thumb-up"],["Other","otherUp","thumb-up"]],[["Hard to understand","hardToUnderstand","thumb-down"],["Incorrect information or sample code","incorrectInformationOrSampleCode","thumb-down"],["Missing the information/samples I need","missingTheInformationSamplesINeed","thumb-down"],["Other","otherDown","thumb-down"]],["Last updated 2026-02-25 UTC."],[],[]]
