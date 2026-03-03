# Gemini API libraries  |  Google AI for Developers

Source: https://ai.google.dev/gemini-api/docs/libraries

Nano Banana 2 is here. [Learn more in our blog](https://blog.google/innovation-and-ai/technology/developers-tools/build-with-nano-banana-2/). 

  * [ Home ](https://ai.google.dev/)
  * [ Gemini API ](https://ai.google.dev/gemini-api)
  * [ Docs ](https://ai.google.dev/gemini-api/docs)



Send feedback 

#  Gemini API libraries

When building with the Gemini API, we recommend using the **Google GenAI SDK**. These are the official, production-ready libraries that we develop and maintain for the most popular languages. They are in [General Availability](https://ai.google.dev/gemini-api/docs/libraries#new-libraries) and used in all our official documentation and examples.

**Note:** If you're using one of our legacy libraries, we strongly recommend you [migrate](https://ai.google.dev/gemini-api/docs/migrate) to the Google GenAI SDK. Review the [legacy libraries](https://ai.google.dev/gemini-api/docs/libraries#previous-sdks) section for more information. If you're using an AI coding assistant, install the [Gemini API development skill](https://ai.google.dev/gemini-api/docs/coding-agents) to give your agent access to the latest documentation and best practices.

If you're new to the Gemini API, follow our [quickstart guide](https://ai.google.dev/gemini-api/docs/quickstart) to get started.

## Language support and installation

The Google GenAI SDK is available for the Python, JavaScript/TypeScript, Go and Java languages. You can install each language's library using package managers, or visit their GitHub repos for further engagement:

### Python

  * Library: [`google-genai`](https://pypi.org/project/google-genai)

  * GitHub Repository: [googleapis/python-genai](https://github.com/googleapis/python-genai)

  * Installation: `pip install google-genai`




### JavaScript

  * Library: [`@google/genai`](https://www.npmjs.com/package/@google/genai)

  * GitHub Repository: [googleapis/js-genai](https://github.com/googleapis/js-genai)

  * Installation: `npm install @google/genai`




### Go

  * Library: [`google.golang.org/genai`](https://pkg.go.dev/google.golang.org/genai)

  * GitHub Repository: [googleapis/go-genai](https://github.com/googleapis/go-genai)

  * Installation: `go get google.golang.org/genai`




### Java

  * Library: `google-genai`

  * GitHub Repository: [googleapis/java-genai](https://github.com/googleapis/java-genai)

  * Installation: If you're using Maven, add the following to your dependencies:



[code] 
    <dependencies>
      <dependency>
        <groupId>com.google.genai</groupId>
        <artifactId>google-genai</artifactId>
        <version>1.0.0</version>
      </dependency>
    </dependencies>
    
[/code]

### C#

  * Library: `Google.GenAI`

  * GitHub Repository: [googleapis/dotnet-genai](https://googleapis.github.io/dotnet-genai/)

  * Installation: `dotnet add package Google.GenAI`




## General availability

As of May 2025, the Google GenAI SDK has reached General Availability (GA) across all supported platforms and are the recommended libraries to access the Gemini API. They are stable, fully supported for production use, and are actively maintained. They provide access to the latest features, and offer the best performance working with Gemini.

If you're using one of our legacy libraries, we strongly recommend you migrate so that you can access the latest features and get the best performance working with Gemini. Review the [legacy libraries](https://ai.google.dev/gemini-api/docs/libraries#previous-sdks) section for more information.

## Legacy libraries and migration

If you are using one of our legacy libraries, we recommend that you [migrate to the new libraries](https://ai.google.dev/gemini-api/docs/migrate).

The legacy libraries don't provide access to recent features (such as [Live API](https://ai.google.dev/gemini-api/docs/live) and [Veo](https://ai.google.dev/gemini-api/docs/video)) and are deprecated as of November 30th, 2025.

Each legacy library's support status varies, detailed in the following table:

Language | Legacy library | Support status | Recommended library  
---|---|---|---  
**Python** | `[google-generativeai](https://github.com/google-gemini/deprecated-generative-ai-python)` | Not actively maintained | `[google-genai](https://github.com/googleapis/python-genai)`  
**JavaScript/TypeScript** | `[@google/generativeai](https://github.com/google-gemini/generative-ai-js)` | Not actively maintained | `[@google/genai](https://github.com/googleapis/js-genai)`  
**Go** | `[google.golang.org/generative-ai](https://github.com/google/generative-ai-go)` | Not actively maintained | `[google.golang.org/genai](https://github.com/googleapis/go-genai)`  
**Dart and Flutter** | `[google_generative_ai](https://pub.dev/packages/google_generative_ai/install)` | Not actively maintained | Use trusted community or third party libraries, like [firebase_ai](https://pub.dev/packages/firebase_ai), or access using REST API  
**Swift** | `[generative-ai-swift](https://github.com/google/generative-ai-swift)` | Not actively maintained | Use [Firebase AI Logic](https://firebase.google.com/products/firebase-ai-logic)  
**Android** | `[generative-ai-android](https://github.com/google-gemini/generative-ai-android)` | Not actively maintained | Use [Firebase AI Logic](https://firebase.google.com/products/firebase-ai-logic)  
  
**Note for Java developers:** There was no legacy Google-provided Java SDK for the Gemini API, so no migration from a previous Google library is required. You can start directly with the new library in the Language support and installation section.

Send feedback 

Except as otherwise noted, the content of this page is licensed under the [Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/), and code samples are licensed under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0). For details, see the [Google Developers Site Policies](https://developers.google.com/site-policies). Java is a registered trademark of Oracle and/or its affiliates.

Last updated 2026-02-24 UTC.

Need to tell us more?  [[["Easy to understand","easyToUnderstand","thumb-up"],["Solved my problem","solvedMyProblem","thumb-up"],["Other","otherUp","thumb-up"]],[["Missing the information I need","missingTheInformationINeed","thumb-down"],["Too complicated / too many steps","tooComplicatedTooManySteps","thumb-down"],["Out of date","outOfDate","thumb-down"],["Samples / code issue","samplesCodeIssue","thumb-down"],["Other","otherDown","thumb-down"]],["Last updated 2026-02-24 UTC."],[],[]]
