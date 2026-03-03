# Secret Manager overview  |  Google Cloud Documentation

Source: https://docs.cloud.google.com/secret-manager/docs/overview

* [ Home ](https://docs.cloud.google.com/)
  * [ Documentation ](https://docs.cloud.google.com/docs)
  * [ Security ](https://docs.cloud.google.com/docs/security)
  * [ Secret Manager ](https://docs.cloud.google.com/secret-manager/docs)
  * [ Guides ](https://docs.cloud.google.com/secret-manager/docs/overview)



Send feedback 

#  Secret Manager overview Stay organized with collections  Save and categorize content based on your preferences. 

Secret Manager is a secrets and credential management service that lets you store and manage sensitive data such as API keys, usernames, passwords, certificates, and more.

A [_secret_](https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets) is a global resource that contains a collection of metadata and secret versions. The metadata can include labels, annotations, and permissions.

A [_secret version_](https://cloud.google.com/secret-manager/docs/add-secret-version) stores the actual secret data, such as API keys, passwords, or certificates. Each version is identified by a unique ID or timestamp.

Using Secret Manager, you can do the following:

  * **Manage rollback, recovery, and auditing using versions** : Versions help you manage gradual rollouts and emergency rollback, If a secret is accidentally changed or compromised, you can revert to a previous, known-good version. This minimizes potential downtime and security breaches. Versioning maintains a historical record of changes made to a secret, including who made the changes and when. It helps you audit secret data and track any unauthorized access attempts. You can pin secret versions to specific workloads and add [aliases](https://cloud.google.com/secret-manager/docs/assign-alias-to-secret-version) for easier access to secret data. You can also [disable](https://cloud.google.com/secret-manager/docs/disable-secret-version) or [destroy](https://cloud.google.com/secret-manager/docs/destroy-secret-version) secret versions that you don't require.

  * **Encrypt your secret data in transit and at rest** : All secrets are encrypted by default, both in transit using TLS and at rest with AES-256-bit encryption keys. For those requiring more granular control, you can encrypt your secret data with [Customer-Managed Encryption Keys (CMEK)](https://cloud.google.com/secret-manager/docs/cmek). Using CMEK, you can generate new encryption keys or import existing ones to meet your specific requirements.

  * **Manage access to secrets using fine-grained Identity and Access Management (IAM) roles and conditions** : With [IAM roles and permissions](https://cloud.google.com/secret-manager/docs/access-control), you can [provide granular access](https://cloud.google.com/secret-manager/docs/manage-access-to-secrets) to specific Secret Manager resources. You can segregate responsibilities for accessing, managing, auditing, and rotating secrets.

  * **Ensure high availability and disaster recovery with secret replication** : You can [replicate your secrets](https://cloud.google.com/secret-manager/docs/choosing-replication) across multiple regions to ensure high availability and disaster recovery for your applications regardless of their geographic location. You can choose between the following replication policies: 
    * [Automatic replication](https://cloud.google.com/secret-manager/docs/choosing-replication#automatic): Google Cloud decides the regions considering availability and latency. You are only charged for one location.

    * [User managed replication](https://cloud.google.com/secret-manager/docs/choosing-replication#user-managed): You can select a custom set of regions depending on your requirements. You are charged per location.

  * **Rotate secrets automatically to meet your security and compliance requirements** : [Rotating your secrets](https://cloud.google.com/secret-manager/docs/rotation-recommendations) protects against unauthorized access and data breaches. Regularly changing your secrets reduces the risk of stale or forgotten secrets and ensures compliance with many regulatory frameworks that require periodic rotation of sensitive credentials.

  * **Enforce data residency using regional secrets** : [Data residency](https://cloud.google.com/architecture/framework/security/meet-regulatory-compliance-and-privacy-needs#control_data_residency) requires that certain types of data, often belonging to specific individuals or organizations, be stored within a defined geographic location. You can create [regional secrets](https://cloud.google.com/secret-manager/docs/create-regional-secrets) and store your sensitive data within a specific location to comply with data sovereignty laws and regulations.

  * **Manage operational parameters for your applications using Parameter Manager** : [Parameter Manager](https://cloud.google.com/secret-manager/parameter-manager/docs/overview) is an extension to the Secret Manager service that you can use to store and manage application configurations such as database connection strings, feature flags, environment names, port numbers to listen on, and settings for application features. You can also [reference secrets](https://cloud.google.com/secret-manager/parameter-manager/docs/reference-secrets-in-parameter) stored in Secret Manager within your parameter configurations. To use Parameter Manager, you must enable the Parameter Manager API and grant your users the [required IAM roles](https://cloud.google.com/secret-manager/parameter-manager/docs/access-control).

## Difference between secrets management and key management

Secrets management and key management are both critical components of data security, but they serve distinct purposes and handle different types of sensitive information. The choice between secrets management and key management depends on your specific needs. If you want to securely store and manage confidential data, a secrets management system is the right tool. If you want to manage encryption keys and perform cryptographic operations, a key management system is the better choice.

You can use the following table to understand the key differences between Secret Manager and a key management system, such as [Cloud Key Management Service(Cloud KMS)](https://cloud.google.com/kms/docs).

Feature | Secret Manager | Cloud KMS  
---|---|---  
Primary function | Store, manage, and access secrets as binary blobs or text strings. | Manage cryptographic keys and use them to encrypt or decrypt data.  
Data stored | Actual secret values. With the appropriate permissions, you can view the contents of the secret. | Cryptographic keys. You can't view, extract, or export the actual cryptographic secrets (the bits and bytes) that are used for encryption and decryption operations.  
Encryption | Encrypts secrets at rest and in transit using Google-owned and managed keys or customer-managed keys. | Provides encryption and decryption capabilities for other services.  
Typical use cases | Store configuration information such as database passwords, API keys, or TLS certificates needed by an application at runtime. | Handle large encryption workloads, such as encrypting rows in a database or encrypting binary data such as images and files. You can also use Cloud KMS to perform other cryptographic operations such as signing and verification.  
  
## Encryption of secrets

Secret Manager always encrypts your secret data before it is persisted to disk. To learn more about Google Cloud encryption options, refer to [Encryption at rest](https://cloud.google.com/docs/security/encryption/default-encryption).

Secret Manager manages server-side encryption keys on your behalf using the same hardened key management systems that we use for our own encrypted data, including strict key access controls and auditing. Secret Manager encrypts user data at rest using AES-256. There is no setup or configuration required, no need to modify the way you access the service, and no visible performance impact. Your secret data is automatically and transparently decrypted when accessed by an authorized user.

The Secret Manager API always communicates over a secure HTTP(S) connection.

Those who require an extra layer of protection can enable CMEK and use their own encryption keys stored in Cloud Key Management Service to protect the secrets stored in Secret Manager. See the [CMEK documentation](https://cloud.google.com/secret-manager/docs/cmek) for details on how to configure and use customer-managed encryption keys.

## What's next

    * Learn how to [create a secret](https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets).
    * Learn how to [add a secret version](https://cloud.google.com/secret-manager/docs/add-secret-version).
    * Learn how to [edit a secret](https://cloud.google.com/secret-manager/docs/edit-secrets).
    * Learn about [quotas and limitations](https://cloud.google.com/secret-manager/quotas).
    * Learn about [best practices](https://cloud.google.com/secret-manager/docs/best-practices).



Send feedback 

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2026-02-25 UTC.

Need to tell us more?  [[["Easy to understand","easyToUnderstand","thumb-up"],["Solved my problem","solvedMyProblem","thumb-up"],["Other","otherUp","thumb-up"]],[["Hard to understand","hardToUnderstand","thumb-down"],["Incorrect information or sample code","incorrectInformationOrSampleCode","thumb-down"],["Missing the information/samples I need","missingTheInformationSamplesINeed","thumb-down"],["Other","otherDown","thumb-down"]],["Last updated 2026-02-25 UTC."],[],[]]
