const express = require("express");
const app = express();
const port = 3002;

app.get("/", (req, res) => {
  res.send("Hello World!");
});

app.get("/api/list-offerings/:namespace/:name", (req, res) => {
  const n = req.params.name;
  const ns = req.params.namespace;
  res.setHeader("Access-Control-Allow-Origin", "*");

  if (ns === "kymasystem" && n === "defaultsecret") {
    res.send({
      num_items: 16,
      items: [
        {
          id: "7dc306e2-c1b5-46b3-8237-bcfbda56ba66",
          ready: true,
          name: "service-manager",
          description:
            "The central registry for service brokers and platforms in SAP Business Technology Platform",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          metadata: {
            createBindingDocumentationUrl:
              "https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/1ca5bbeac19340ce959e82b51b2fde1e.html",
            discoveryCenterUrl:
              "https://discovery-center.cloud.sap/serviceCatalog/service-management",
            displayName: "Service Manager",
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/f13b6c63eef341bc8b7d25b352401c92.html",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMjI5IiBkYXRhLW5hbWU9IkxheWVyIDIyOSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIiB2aWV3Qm94PSIwIDAgNTYgNTYiPjxkZWZzPjxzdHlsZT4uY2xzLTF7ZmlsbDojMGE2ZWQxO30uY2xzLTJ7ZmlsbDojMDUzYjcwO308L3N0eWxlPjwvZGVmcz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0yOCw3YTMsMywwLDEsMS0zLDMsMywzLDAsMCwxLDMtM20wLTNhNiw2LDAsMSwwLDYsNiw2LjAwNyw2LjAwNywwLDAsMC02LTZaIi8+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNMjgsNDNhMywzLDAsMSwxLTMsMywzLDMsMCwwLDEsMy0zbTAtM2E2LDYsMCwxLDAsNiw2LDYuMDA3LDYuMDA3LDAsMCwwLTYtNloiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMywyNXY2SDdWMjVoNm0zLTNINFYzNEgxNlYyMloiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00OSwyNXY2SDQzVjI1aDZtMy0zSDQwVjM0SDUyVjIyWiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTM3LDI2LjEyNUE3LjEzMyw3LjEzMywwLDAsMSwyOS44NzUsMTlhMS4xMjUsMS4xMjUsMCwwLDEsMi4yNSwwQTQuODc5LDQuODc5LDAsMCwwLDM3LDIzLjg3NWExLjEyNSwxLjEyNSwwLDAsMSwwLDIuMjVaIi8+PHBhdGggY2xhc3M9ImNscy0yIiBkPSJNMTksMjYuMTI1YTEuMTI1LDEuMTI1LDAsMCwxLDAtMi4yNUE0Ljg3OSw0Ljg3OSwwLDAsMCwyMy44NzUsMTlhMS4xMjUsMS4xMjUsMCwwLDEsMi4yNSwwQTcuMTMzLDcuMTMzLDAsMCwxLDE5LDI2LjEyNVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNSwzOC4xMjVBMS4xMjUsMS4xMjUsMCwwLDEsMjMuODc1LDM3LDQuODgsNC44OCwwLDAsMCwxOSwzMi4xMjVhMS4xMjUsMS4xMjUsMCwwLDEsMC0yLjI1QTcuMTMzLDcuMTMzLDAsMCwxLDI2LjEyNSwzNywxLjEyNSwxLjEyNSwwLDAsMSwyNSwzOC4xMjVaIi8+PHBhdGggY2xhc3M9ImNscy0yIiBkPSJNMzEsMzguMTI1QTEuMTI1LDEuMTI1LDAsMCwxLDI5Ljg3NSwzNyw3LjEzMyw3LjEzMywwLDAsMSwzNywyOS44NzVhMS4xMjUsMS4xMjUsMCwwLDEsMCwyLjI1QTQuODgsNC44OCwwLDAsMCwzMi4xMjUsMzcsMS4xMjUsMS4xMjUsMCwwLDEsMzEsMzguMTI1WiIvPjwvc3ZnPg==",
            longDescription:
              "SAP Service Manager allows you to consume platform services in any connected runtime environment, track service instances creation, and share services and service instances between different environments.",
            serviceInventoryId: "SERVICE-324",
            shareable: true,
            supportUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/5dd739823b824b539eee47b7860a00be.html",
          },
          broker_id: "c7788cfd-3bd9-4f66-9bd8-487a23142f2c",
          catalog_id: "6e6cc910-c2f7-4b95-a725-c986bb51bad7",
          catalog_name: "service-manager",
          created_at: "2020-08-09T11:31:20.082571Z",
          updated_at: "2024-04-03T15:02:08.45919Z",
        },
        {
          id: "b3f88a98-4076-4d8b-b519-1c5222c9b178",
          ready: true,
          name: "lps-service",
          description: "Service for integrating with LPS Service",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: true,
          allow_context_updates: false,
          tags: ["lps", "service"],
          metadata: {
            shareable: false,
            displayName: "LPS Service",
          },
          broker_id: "bd821762-2eb0-407a-8b09-d80330750d1d",
          catalog_id: "72d71e2f-aa3e-4fc9-bfc0-8a5a8b541570",
          catalog_name: "lps-service",
          created_at: "2020-08-10T07:34:28.809068Z",
          updated_at: "2024-04-03T15:02:34.107748Z",
        },
        {
          id: "a5387c0b-141b-4b66-bb14-9fdb032e6eaf",
          ready: true,
          name: "saas-registry",
          description:
            "Service for application providers to register multitenant applications and services",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["SaaS"],
          metadata: {
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/5e8a2b74e4f2442b8257c850ed912f48.html",
            serviceInventoryId: "SERVICE-859",
            displayName: "SaaS Provisioning Service",
            imageUrl:
              "data:image/png;base64,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",
          },
          broker_id: "e1c79edb-21eb-4b15-b873-176fc64cc438",
          catalog_id: "lps-saas-registry-service-broker",
          catalog_name: "saas-registry",
          created_at: "2020-08-10T07:35:37.447784Z",
          updated_at: "2024-04-03T15:02:40.702428Z",
        },
        {
          id: "8627a19b-c397-4b1a-b297-6281bd46d8c3",
          ready: true,
          name: "destination",
          description:
            "Provides a secure and reliable access to destination and certificate configurations",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["destination", "conn", "connsvc"],
          metadata: {
            longDescription:
              "Use the Destination service to provide your cloud applications with access to destination and certificate configurations in a secure and reliable way",
            documentationUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/34010ace6ac84574a4ad02f5055d3597.html",
            providerDisplayName: "SAP SE",
            serviceInventoryId: "SERVICE-171",
            displayName: "Destination",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iZGVzdGluYXRpb24iIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDU2IDU2Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6IzVhN2E5NDt9LmNscy0ye2ZpbGw6IzAwOTJkMTt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPmRlc3RpbmF0aW9uPC90aXRsZT48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xOSw1MkgxMC4wOTRhMy4wNzIsMy4wNzIsMCwwLDEtMi4yLS44NDRBMi44MzcsMi44MzcsMCwwLDEsNyw0OVYxNkwxOSw0SDQwYTIuODQxLDIuODQxLDAsMCwxLDIuMTU2Ljg5MUEyLjk2MiwyLjk2MiwwLDAsMSw0Myw3djNINDBWN0gyMnY5YTIuODQ0LDIuODQ0LDAsMCwxLS44OTEsMi4xNTZBMi45NjIsMi45NjIsMCwwLDEsMTksMTlIMTBWNDloOVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNy45MzgsNDEuODYzLDI0LjcxNiw0MC4ybC0yLjAyNCwxLjg1OUwyMC4xMTUsMzkuNTJsMS43NjQtMS43NjQtMS4zNjctMy41MjdMMTgsMzQuMmwwLTMuNTc2aDIuNDc5bDEuNDctMy41NTEtMS44MzQtMS44NDUsMi41My0yLjU3NCwxLjkxMiwxLjkxMSwzLjM4MS0xLjQtLjAxNS0yLjc1NCwzLjc2NS4wMTd2Mi43MzdsMy4zOCwxLjRMMzcuMDg0LDIyLjgsMzkuNTEsMjUuNDhsLTEuNzY0LDEuNzY0LDEuNCwzLjM4MSwyLjY2Ni4xODdWMzIuNWgzVjMwLjgxMmEzLjEyNSwzLjEyNSwwLDAsMC0zLjE4OC0zLjE4N2gtLjAybC4wODItLjA3OWEzLjI3NSwzLjI3NSwwLDAsMCwuODU4LTIuMjE4LDMuMDc2LDMuMDc2LDAsMCwwLS45MTQtMi4yMjFsLTIuNDI2LTIuNDI1YTMuMjYxLDMuMjYxLDAsMCwwLTQuNDk0LDBsLS4wMjMuMDIzdi0uMDE3QTMuMTI1LDMuMTI1LDAsMCwwLDMxLjUsMTcuNUgyOC4xMjVhMy4xMjMsMy4xMjMsMCwwLDAtMy4xODcsMy4xODh2LjAxN2wtLjAyNC0uMDIzYTMuMjYxLDMuMjYxLDAsMCwwLTQuNDk0LDBsLTIuNDI2LDIuNDI1YTMuMDgsMy4wOCwwLDAsMC0uOTE0LDIuMjIxLDMuMzA5LDMuMzA5LDAsMCwwLC45MTQsMi4yNzRsLjAyNC4wMjNIMThhMy4xMjMsMy4xMjMsMCwwLDAtMy4xODcsMy4xODd2My4zNzZhMy4xNzcsMy4xNzcsMCwwLDAsLjg4NCwyLjIxNywzLjA4OCwzLjA4OCwwLDAsMCwyLjMuOTdoLjAxOGwtLjAyNC4wMjNhMy4yMiwzLjIyLDAsMCwwLDAsNC40OTVsMi40MjYsMi40MjVhMy4yNDUsMy4yNDUsMCwwLDAsNC41MTgtLjAyM3YuMDE3YTMuMTc4LDMuMTc4LDAsMCwwLC44ODQsMi4yMTgsMy4wODgsMy4wODgsMCwwLDAsMi4zLjk3aDEuNjg4di0zbC0xLjg3NS0uMTg4WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTI5LjgxMywyOS41QTIuOTU4LDIuOTU4LDAsMCwxLDMyLjM1MiwzMUgzNS42YTUuOTg3LDUuOTg3LDAsMSwwLTcuMjg2LDcuMjg3VjM1LjAzOWEyLjk1NiwyLjk1NiwwLDAsMS0xLjUtMi41MzlBMywzLDAsMCwxLDI5LjgxMywyOS41WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTQzLjg2OSw0NS4yNzhsLjI2NC0uMjY1YTQuNTE0LDQuNTE0LDAsMCwwLDAtNi4zNjVMNDAuNzgxLDM1LjNhNC41MTYsNC41MTYsMCwwLDAtNi4zNjYsMGwtLjI2NC4yNjUtMy4xNjctMy4xNjctMS41OTEsMS41OTEsMy4xNjcsMy4xNjctLjI2NS4yNjRhNC41MTYsNC41MTYsMCwwLDAsMCw2LjM2NmwzLjM1MywzLjM1MmE0LjUxNSw0LjUxNSwwLDAsMCw2LjM2NSwwbC4yNjUtLjI2NEw0Ny40MDksNTIsNDksNTAuNDA5Wk0zNC42NDEsNDMuMmwtLjctLjdhMi40LDIuNCwwLDAsMSwwLTMuMzgxbDIuMTc3LTIuMTc2YTIuNCwyLjQsMCwwLDEsMy4zOCwwbC43LjdabTcuODQ0LjExLTIuMTc3LDIuMTc2YTIuNCwyLjQsMCwwLDEtMy4zOCwwbC0uNy0uNyw1LjU1Ny01LjU1Ny43LjdBMi40LDIuNCwwLDAsMSw0Mi40ODUsNDMuMzA4WiIvPjwvc3ZnPg==",
            supportUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/e5580c5dbb5710149e53c6013301a9f2.html",
          },
          broker_id: "624a27b3-14b6-4317-a71e-5506896d0ce4",
          catalog_id: "a8683418-15f9-11e7-873e-02667c123456",
          catalog_name: "destination",
          created_at: "2020-08-10T14:58:38.756598Z",
          updated_at: "2024-04-03T15:03:15.652954Z",
        },
        {
          id: "547b140d-9dfc-469c-85b1-0680c14ce1be",
          ready: true,
          name: "connectivity",
          description:
            "Establishes a secure and reliable connectivity between cloud applications and on-premise systems.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: true,
          allow_context_updates: false,
          tags: ["connectivity", "conn", "connsvc"],
          metadata: {
            longDescription:
              "Use the Connectivity service to establish secure and reliable connectivity between your cloud applications and on-premise systems running in isolated networks.",
            documentationUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/34010ace6ac84574a4ad02f5055d3597.html",
            providerDisplayName: "SAP SE",
            serviceInventoryId: "SERVICE-169",
            displayName: "Connectivity",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0ic2FwLWhhbmEtY2xvdWQtY29ubmVjdG9yIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1NiA1NiI+PGRlZnM+PHN0eWxlPi5jbHMtMXtmaWxsOiMwMDkyZDE7fS5jbHMtMntmaWxsOiM1YTdhOTQ7fTwvc3R5bGU+PC9kZWZzPjx0aXRsZT5zYXAtaGFuYS1jbG91ZC1jb25uZWN0b3I8L3RpdGxlPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTQxLjUsNDloLTlhMS41LDEuNSwwLDAsMCwwLDNoOWExLjUsMS41LDAsMCwwLDAtM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00OC45OTEsMjVIMjUuMDA5QTMuMDA5LDMuMDA5LDAsMCwwLDIyLDI4LjAwOVY0Mi45OTFBMy4wMDksMy4wMDksMCwwLDAsMjUuMDA5LDQ2SDQ4Ljk5MUEzLjAwOSwzLjAwOSwwLDAsMCw1Miw0Mi45OTFWMjguMDA5QTMuMDA5LDMuMDA5LDAsMCwwLDQ4Ljk5MSwyNVptMCwxOEwyNSw0Mi45OTEsMjUuMDA5LDI4SDQ4Ljk5MWwuMDA5LjAwOVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0xOS4xMDksN2E2LjQ1Nyw2LjQ1NywwLDAsMSw1Ljg2NSw0LjAzNGwxLjMwNiwzLjI4OUwyOS4zMSwxMi41YTMuOTE5LDMuOTE5LDAsMCwxLDIuMDQzLS41OTEsMy45ODcsMy45ODcsMCwwLDEsMy45MTQsMy4yNDlsLjI4OCwxLjUyOSwxLjQxNS42NDVhNS4zNTEsNS4zNTEsMCwwLDEsMyw0LjY3SDQzYTguMzU2LDguMzU2LDAsMCwwLTQuNzg1LTcuNEE2Ljk0MSw2Ljk0MSwwLDAsMCwyNy43NjIsOS45MjgsOS40NDksOS40NDksMCwwLDAsMTkuMDU1LDRDOC43LDQuNTQ4LDkuOCwxNC42MjEsOS44LDE0LjYyMUE4LjM4Nyw4LjM4NywwLDAsMCwxMi40MSwzMC45ODZIMTl2LTNIMTIuNDFhNS4zODcsNS4zODcsMCwwLDEtMS42NzUtMTAuNTE1bDIuMzA4LS43NTlMMTIuNzgxLDE0LjNhOC4xMSw4LjExLDAsMCwxLDEuNS01LjI4NEE2LjUsNi41LDAsMCwxLDE5LjEwOSw3WiIvPjwvc3ZnPg==",
            supportUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/e5580c5dbb5710149e53c6013301a9f2.html",
          },
          broker_id: "e453233d-64e9-46c7-a1fd-ee164f561309",
          catalog_id: "7e2071bd-3e15-4839-8615-c6adf8d58ad0",
          catalog_name: "connectivity",
          created_at: "2020-08-10T16:46:27.305722Z",
          updated_at: "2024-04-03T15:03:20.801493Z",
        },
        {
          id: "70da63ba-36c0-4f5b-8b64-63e02e501d44",
          ready: true,
          name: "metering-service",
          description:
            "Record usage data for commercial purposes like billing, charging, and resource planning.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: true,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["metering", "reporting"],
          metadata: {
            documentationUrl:
              "https://int.controlcenter.ondemand.com/index.html#/knowledge_center/articles/879701d81a314fe59a1ae48c56ab2526",
            serviceInventoryId: "SERVICE-367",
            displayName: "Metering Service",
          },
          broker_id: "967da469-6e7b-4d6e-ba9b-e5c32ce5027d",
          catalog_id: "metering-service-broker",
          catalog_name: "metering-service",
          created_at: "2020-08-12T13:15:46.933069Z",
          updated_at: "2024-04-03T15:03:30.855055Z",
        },
        {
          id: "d67ff82d-9bfe-43e3-abd2-f2e21a5362c5",
          ready: true,
          name: "xsuaa",
          description:
            "Manage application authorizations and trust to identity providers.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["xsuaa"],
          metadata: {
            longDescription:
              "Configure trust to identity providers for authentication. Manage your authorization model consisting of roles, groups and role collections, and assigning them to users. Use RESTful APIs to automate and integrate with other systems.",
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/6373bb7a96114d619bfdfdc6f505d1b9.html",
            serviceInventoryId: "SERVICE-92",
            displayName: "Authorization and Trust Management Service",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iYXV0aG9yaXphdGlvbi1tYW5hZ2VtZW50IiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1NiA1NiI+PGRlZnM+PHN0eWxlPi5jbHMtMXtmaWxsOiM1YTdhOTQ7fS5jbHMtMntmaWxsOiMwMDkyZDE7fTwvc3R5bGU+PC9kZWZzPjx0aXRsZT5hdXRob3JpemF0aW9uLW1hbmFnZW1lbnQ8L3RpdGxlPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTM1LjIyMSwxMy4xNDFsLjAzOS0zLjUxNmE0Ljk3OCw0Ljk3OCwwLDAsMSwuNDg4LTIuMTg3QTUuNzE0LDUuNzE0LDAsMCwxLDM3LjA3Niw1LjY2YTYuMzY1LDYuMzY1LDAsMCwxLDEuOTkyLTEuMjExQTYuNjY5LDYuNjY5LDAsMCwxLDQxLjUxLDRhNi41MTksNi41MTksMCwwLDEsMi40MjIuNDQ5QTYuNzE4LDYuNzE4LDAsMCwxLDQ1LjkyNCw1LjY2YTUuNjA5LDUuNjA5LDAsMCwxLDEuMzQ4LDEuNzc3LDUsNSwwLDAsMSwuNDg4LDIuMTg4djMuNTE2TTM2Ljk3MSwxMi43NUg0Ni4wMVY5LjYwNWEzLjY0MiwzLjY0MiwwLDAsMC0xLjUtMi45OTQsNC4xNzYsNC4xNzYsMCwwLDAtMy0xLjExMSw0LjE1LDQuMTUsMCwwLDAtMywuOTEyLDQuMDE3LDQuMDE3LDAsMCwwLTEuNSwzLjE5M1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00OC42NTgsMTQuMDJhMi40LDIuNCwwLDAsMC0uOTA4LS44NzlBNC40LDQuNCwwLDAsMCw0NiwxMi43NUgzNi45NjFhNC4zMjQsNC4zMjQsMCwwLDAtMS43NS4zOTEsMi40OTIsMi40OTIsMCwwLDAtLjg3OS44NzlBMi40NTYsMi40NTYsMCwwLDAsMzQsMTUuMjg5VjIxLjVBMi40NjgsMi40NjgsMCwwLDAsMzYuNSwyNGgxMGEyLjQ0MSwyLjQ0MSwwLDAsMCwxLjc1OC0uNzIzQTIuMzg2LDIuMzg2LDAsMCwwLDQ5LDIxLjVWMTUuMjg5QTIuMzUxLDIuMzUxLDAsMCwwLDQ4LjY1OCwxNC4wMlpNNDIuNSwxNy44MzR2Mi45MzFhLjgzMS44MzEsMCwwLDEtMS42NjMsMFYxNy44MzRhMS41MzMsMS41MzMsMCwwLDEtLjY1Ni0xLjI2OSwxLjQ4OCwxLjQ4OCwwLDAsMSwyLjk3NSwwQTEuNTMzLDEuNTMzLDAsMCwxLDQyLjUsMTcuODM0WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTMxLjM2MywzNi42MzdBOS4wNjYsOS4wNjYsMCwwLDAsMjguNDgsMzQuNyw4LjgxMyw4LjgxMywwLDAsMCwyNSwzNEgxNmE4LjczMiw4LjczMiwwLDAsMC0zLjUxNi43LDkuMTQ4LDkuMTQ4LDAsMCwwLTIuODQ4LDEuOTM0QTkuMDMsOS4wMywwLDAsMCw3LjcsMzkuNTIsOC43OTQsOC43OTQsMCwwLDAsNyw0M3Y5SDM0VjQzYTguODEzLDguODEzLDAsMCwwLS43LTMuNDhBOS4wNjYsOS4wNjYsMCwwLDAsMzEuMzYzLDM2LjYzN1pNMzEsNDlIMTBWNDNhNS43NzMsNS43NzMsMCwwLDEsLjQ2NC0yLjMwNyw2LDYsMCwwLDEsMS4yOTQtMS45MzUsNi4xMTYsNi4xMTYsMCwwLDEsMS45MjEtMS4zQTUuNzEyLDUuNzEyLDAsMCwxLDE2LDM3aDlhNS43ODQsNS43ODQsMCwwLDEsMi4zLjQ2Myw1Ljk3OCw1Ljk3OCwwLDAsMSwzLjIzMSwzLjIyOUE1Ljc5Miw1Ljc5MiwwLDAsMSwzMSw0M1oiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNi44NjMsMzEuMzYzQTkuMTQ4LDkuMTQ4LDAsMCwwLDI4LjgsMjguNTE2YTkuMDUzLDkuMDUzLDAsMCwwLDAtN0E4Ljk3Niw4Ljk3NiwwLDAsMCwyMy45OCwxNi43YTkuMDUzLDkuMDUzLDAsMCwwLTcsMCw5LjE0OCw5LjE0OCwwLDAsMC0yLjg0OCwxLjkzNEE5LjAzLDkuMDMsMCwwLDAsMTIuMiwyMS41MmE5LjA1Myw5LjA1MywwLDAsMCwwLDdBOS4xNjUsOS4xNjUsMCwwLDAsMTYuOTg0LDMzLjNhOS4wNTMsOS4wNTMsMCwwLDAsNywwQTkuMDMsOS4wMywwLDAsMCwyNi44NjMsMzEuMzYzWk0yMC41LDMxYTUuNyw1LjcsMCwwLDEtMi4zMjItLjQ1NSw2LjE2Myw2LjE2MywwLDAsMS0zLjIyNC0zLjIyN0E1LjcsNS43LDAsMCwxLDE0LjUsMjVhNS43NzMsNS43NzMsMCwwLDEsLjQ2NC0yLjMwNyw2LDYsMCwwLDEsMS4yOTQtMS45MzUsNi4xMTYsNi4xMTYsMCwwLDEsMS45MjEtMS4zQTUuNzEyLDUuNzEyLDAsMCwxLDIwLjUsMTlhNS43ODQsNS43ODQsMCwwLDEsMi4zLjQ2Myw1Ljk3OCw1Ljk3OCwwLDAsMSwzLjIzMSwzLjIyOUE1Ljc5Miw1Ljc5MiwwLDAsMSwyNi41LDI1YTUuNzEzLDUuNzEzLDAsMCwxLS40NTQsMi4zMTksNi4xMTYsNi4xMTYsMCwwLDEtMS4zLDEuOTIzLDYsNiwwLDAsMS0xLjkzNywxLjI5NUE1Ljc3MSw1Ljc3MSwwLDAsMSwyMC41LDMxWiIvPjwvc3ZnPg==",
          },
          broker_id: "c1ecf1d2-0b7e-412c-901c-c4f678fd6348",
          catalog_id: "xsuaa",
          catalog_name: "xsuaa",
          created_at: "2020-08-13T15:09:38.643826Z",
          updated_at: "2024-04-03T15:03:45.486538Z",
        },
        {
          id: "8d5d96d0-fa2d-40c9-951f-c9ed571ba5da",
          ready: true,
          name: "feature-flags",
          description: "Feature Flags service for controlling feature rollout",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["feature-flags"],
          metadata: {
            longDescription:
              "Feature Flags service allows you to enable or disable new features at application runtime, based on predefined rules or release plan schedule.",
            documentationUrl:
              "https://help.sap.com/viewer/2250efa12769480299a1acd282b615cf/Cloud/en-US/",
            providerDisplayName: "SAP",
            serviceInventoryId: "SERVICE-172",
            displayName: "Feature Flags",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iZmVhdHVyZWZsYWdzIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1NiA1NiI+PGRlZnM+PHN0eWxlPi5jbHMtMXtmaWxsOiM1YTdhOTQ7fS5jbHMtMntmaWxsOiMwMDkyZDE7fTwvc3R5bGU+PC9kZWZzPjx0aXRsZT5mZWF0dXJlZmxhZ3M8L3RpdGxlPjxjaXJjbGUgY2xhc3M9ImNscy0xIiBjeD0iMzciIGN5PSIxNy41IiByPSI0LjUiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0xOSwyNi41SDM3YTksOSwwLDAsMCwwLTE4SDE5YTksOSwwLDAsMCwwLDE4Wm0xOC0xNWE2LDYsMCwxLDEtNiw2QTYsNiwwLDAsMSwzNywxMS41WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTM3LDI5LjVIMTlhOSw5LDAsMCwwLDAsMThIMzdhOSw5LDAsMCwwLDAtMThaTTM3LDQ2SDE5YTcuNSw3LjUsMCwwLDEsMC0xNUgzN2E3LjUsNy41LDAsMCwxLDAsMTVaIi8+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNMTksMzIuNWE2LDYsMCwxLDAsNiw2QTYsNiwwLDAsMCwxOSwzMi41Wk0xOSw0M2E0LjUsNC41LDAsMSwxLDQuNS00LjVBNC41MDUsNC41MDUsMCwwLDEsMTksNDNaIi8+PC9zdmc+",
            supportUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/5dd739823b824b539eee47b7860a00be.html",
          },
          broker_id: "afe404eb-bab7-4748-a302-ebbf64f56a65",
          catalog_id: "08418a7a-002e-4ff9-b66a-d03fc3d56b16",
          catalog_name: "feature-flags",
          created_at: "2020-08-17T09:00:26.04656Z",
          updated_at: "2024-04-03T15:03:55.667488Z",
        },
        {
          id: "23f7803c-57e2-419e-95c3-ea1c86ed2c68",
          ready: true,
          name: "html5-apps-repo",
          description:
            "Enables storage of HTML5 applications and provides runtime environment for HTML5 applications.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: [
            "html5appsrepo",
            "html5-apps-repo-rt",
            "html5-apps-rt",
            "html5-apps-repo-dt",
            "html5-apps-dt",
          ],
          metadata: {
            displayName: "HTML5 Application Repository",
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/11d77aa154f64c2e83cc9652a78bb985.html",
            longDescription:
              "The HTML5 Application Repository service enables central storage of HTML5 applications in SAP BTP. In runtime, the service enables the consuming application, typically the application router, to access HTML5 application static content in a secure and efficient manner.",
            supportUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/9220a2fd35d84c888c0ae870ca62bfb7.html",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iaHRtbDUtYXBwbGljYXRpb25zIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1NiA1NiI+PGRlZnM+PHN0eWxlPi5jbHMtMXtmaWxsOiMwNTNiNzA7fS5jbHMtMntmaWxsOiMwYTZlZDE7fTwvc3R5bGU+PC9kZWZzPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTQyLjMsMTlhOC4wMDgsOC4wMDgsMCwwLDAtNC4wNzgtNC40QTYuOTQ0LDYuOTQ0LDAsMCwwLDI3Ljc2OSw5LjkyOCw5LjQ1Myw5LjQ1MywwLDAsMCwxOS4wNiw0QzkuMDc4LDQsOS44LDE0LjYyMSw5LjgsMTQuNjIxYTguMzg3LDguMzg3LDAsMCwwLDIuNjEzLDE2LjM2NUgyOC4wMDd2LTNIMTIuNDEzYTUuMzg3LDUuMzg3LDAsMCwxLTEuNjc2LTEwLjUxNWwyLjMwOS0uNzU5TDEyLjc4MywxNC4zYTguMTE0LDguMTE0LDAsMCwxLDEuNS01LjI4NCw2LjQ4NCw2LjQ4NCwwLDAsMSwxMC43LDIuMDIybDEuMzA3LDMuMjlMMjkuMzE4LDEyLjVhMy45MjMsMy45MjMsMCwwLDEsMi4wNDQtLjU5MSwzLjk4OCwzLjk4OCwwLDAsMSwzLjkxNCwzLjI0OWwuMjg5LDEuNTI5LDEuNDE1LjY0NkE0LjgsNC44LDAsMCwxLDM4LjkzMywxOVoiLz48cG9seWdvbiBjbGFzcz0iY2xzLTIiIHBvaW50cz0iMzQuMDcgMjQuNjkxIDM1LjMwOCAyNC42OTEgMzUuMzA4IDI2LjA0NiAzNi42NiAyNi4wNDYgMzYuNjYgMjIgMzUuMzA4IDIyIDM1LjMwOCAyMy4zMzYgMzQuMDcgMjMuMzM2IDM0LjA3IDIyIDMyLjcxOCAyMiAzMi43MTggMjYuMDQ2IDM0LjA3IDI2LjA0NiAzNC4wNyAyNC42OTEiLz48cG9seWdvbiBjbGFzcz0iY2xzLTIiIHBvaW50cz0iMzguNDM5IDI2LjA0NiAzOS43OTIgMjYuMDQ2IDM5Ljc5MiAyMy4zNDIgNDAuOTgzIDIzLjM0MiA0MC45ODMgMjIgMzcuMjQ4IDIyIDM3LjI0OCAyMy4zNDIgMzguNDM5IDIzLjM0MiAzOC40MzkgMjYuMDQ2Ii8+PHBvbHlnb24gY2xhc3M9ImNscy0yIiBwb2ludHM9IjQyLjg5OSAyNC4wNCA0My44MyAyNS40NzkgNDMuODU0IDI1LjQ3OSA0NC43ODQgMjQuMDQgNDQuNzg0IDI2LjA0NiA0Ni4xMzEgMjYuMDQ2IDQ2LjEzMSAyMiA0NC43MiAyMiA0My44NTQgMjMuNDIxIDQyLjk4NiAyMiA0MS41NzYgMjIgNDEuNTc2IDI2LjA0NiA0Mi44OTkgMjYuMDQ2IDQyLjg5OSAyNC4wNCIvPjxwb2x5Z29uIGNsYXNzPSJjbHMtMiIgcG9pbnRzPSI1MC4wNTkgMjQuNzA4IDQ4LjE1NyAyNC43MDggNDguMTU3IDIyIDQ2LjgwNCAyMiA0Ni44MDQgMjYuMDQ2IDUwLjA1OSAyNi4wNDYgNTAuMDU5IDI0LjcwOCIvPjxwb2x5Z29uIGNsYXNzPSJjbHMtMiIgcG9pbnRzPSIzNi4xNyAzNC40OTEgMzYuNjg1IDQwLjI2OCA0MS4zNjMgNDAuMjY4IDQxLjM3NyA0MC4yNjggNDMuOTQ1IDQwLjI2OCA0My43MDIgNDIuOTg2IDQxLjM2MyA0My42MTcgNDEuMzYzIDQzLjYxOCA0MS4zNjEgNDMuNjE4IDM5LjAyNiA0Mi45ODggMzguODc2IDQxLjMxNiAzNy43NDIgNDEuMzE2IDM2Ljc3MSA0MS4zMTYgMzcuMDY1IDQ0LjYwNyA0MS4zNjEgNDUuNzk5IDQxLjM3IDQ1Ljc5NiA0MS4zNyA0NS43OTYgNDUuNjYyIDQ0LjYwNyA0NS42OTMgNDQuMjUzIDQ2LjE4NiAzOC43MzUgNDYuMjM3IDM4LjE3MiA0NS42NzIgMzguMTcyIDQxLjM3NyAzOC4xNzIgNDEuMzYzIDM4LjE3MiAzOC42MDMgMzguMTcyIDM4LjQxMSAzNi4wMjUgNDEuMzcgMzYuMDI1IDQxLjM3NyAzNi4wMjUgNDYuNDI4IDM2LjAyNSA0Ni40MzUgMzYuMDI1IDQ2LjQ3NyAzNS41NTQgNDYuNTczIDM0LjQ5MSA0Ni42MjMgMzMuOTI5IDQxLjM3NyAzMy45MjkgNDEuMzcgMzMuOTI5IDM2LjEyIDMzLjkyOSAzNi4xNyAzNC40OTEiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0zMC43NCwyNy45LDMyLjY3NCw0OS41OSw0MS4zNTcsNTJsOC43MDYtMi40MTNMNTIsMjcuOVpNNDcuNjg2LDQ3LjM1OCw0MS4zNyw0OS4xMDlsLTYuMzE2LTEuNzUxTDMzLjU2NywzMC43MTZoMTUuNloiLz48L3N2Zz4=",
            serviceInventoryId: "SERVICE-234",
          },
          broker_id: "e5e75ccc-7963-42cc-b4d1-1314f5ddc6f3",
          catalog_id: "14f042c6-0175-43ef-9a5d-33bd10890e2a",
          catalog_name: "html5-apps-repo",
          created_at: "2020-08-18T16:05:37.292133Z",
          updated_at: "2024-04-03T15:04:05.623748Z",
        },
        {
          id: "0091024c-1648-4716-bd17-604eabd7f480",
          ready: true,
          name: "auditlog-management",
          description: "Retrieve logs and change retention",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          metadata: {
            displayName: "Auditlog Management",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iYXVkaXRsb2ctbWFuYWdlbWVudCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIiB2aWV3Qm94PSIwIDAgNTYgNTYiPjxkZWZzPjxzdHlsZT4uY2xzLTF7ZmlsbDojNWE3YTk0O30uY2xzLTJ7ZmlsbDojMDA5MmQxO308L3N0eWxlPjwvZGVmcz48dGl0bGU+YXVkaXRsb2ctbWFuYWdlbWVudDwvdGl0bGU+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNNDAuNjA4LDEwLjg0M0EyLjk3LDIuOTcsMCwwLDAsMzguNSwxMEgzMi4yMThhNS4yNzYsNS4yNzYsMCwwLDAtMS4zNTktMS45MjEsNC4xLDQuMSwwLDAsMC0yLjItLjk4NSw1Ljg1Miw1Ljg1MiwwLDAsMC0yLjEwOS0yLjI0OUE1LjY2MSw1LjY2MSwwLDAsMCwyMy41LDRhNS45LDUuOSwwLDAsMC0zLjA5My44NDQsNS43MjEsNS43MjEsMCwwLDAtMi4xNTYsMi4yNDksNC4yNzEsNC4yNzEsMCwwLDAtMi4xNTYuOTg1QTQuMjIyLDQuMjIyLDAsMCwwLDE0Ljc4MywxMEg4LjVhMi44ODgsMi44ODgsMCwwLDAtMywzVjQ5YTIuODg4LDIuODg4LDAsMCwwLDMsM2gyN1Y0OUg4LjVWMTNoNi4yODFhNi41MTYsNi41MTYsMCwwLDAsLjkzNywxLjg3NUEzLjAxOCwzLjAxOCwwLDAsMCwxOC4xNTcsMTZIMjguODQ0YTIuOTMsMi45MywwLDAsMCwyLjM0My0xLjEyNUE0LjY0OCw0LjY0OCwwLDAsMCwzMi4yMTgsMTNIMzguNVYyNWgzVjEzQTIuODQ2LDIuODQ2LDAsMCwwLDQwLjYwOCwxMC44NDNaTTI4LDEzSDE5YTEuMzI1LDEuMzI1LDAsMCwxLTEuNS0xLjVBMS4zMjUsMS4zMjUsMCwwLDEsMTksMTBoMS41YTIuODg2LDIuODg2LDAsMCwxLDMtMywyLjk3LDIuOTcsMCwwLDEsMi4xMS44NDNBMi44NTEsMi44NTEsMCwwLDEsMjYuNSwxMEgyOGExLjMyNywxLjMyNywwLDAsMSwxLjUsMS41QTEuMzI2LDEuMzI2LDAsMCwxLDI4LDEzWiIvPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTM3LjkyOCwzMy44NzdjLS4xMDYtLjIzLS4yMzEtLjQ2OS0uMzcyLS43MTdsMS4yNzUtMS4yNzRMMzcuNjEsMzAuNjY1bC0xLjI3NSwxLjI3NEE2LjQ2Myw2LjQ2MywwLDAsMCwzNC44LDMxLjNWMjkuNUgzMy4xdjEuOGE3Ljc0Nyw3Ljc0NywwLDAsMC0xLjQzNC42MzdsLTEuMzI3LTEuMjc0LTEuMTY4LDEuMjIxLDEuMjc0LDEuMjc0YTMuMzc1LDMuMzc1LDAsMCwwLS42MzcsMS40ODdIMjh2MS43aDEuOGEzLjUyLDMuNTIsMCwwLDAsLjYzNywxLjQ4NkwyOS4xNjgsMzkuMTZsMS4xNjgsMS4xNjgsMS4zMjctMS4yNzRhMy41MDksMy41MDksMCwwLDAsMS40MzQuNjM2VjQxLjVoMS43VjM5LjY5YTQuNDU0LDQuNDU0LDAsMCwwLDEuNTM5LS42MzZsMS4yNzUsMS4yNzQsMS4yMjEtMS4xNjgtMS4yNzUtMS4zMjhhNS44NjksNS44NjksMCwwLDAsLjYzOC0xLjQ4Nkg0MHYtMS43aC0xLjhBNC41MDgsNC41MDgsMCwwLDAsMzcuOTI4LDMzLjg3N1pNMzUuOCwzNy4zMjhhMi41LDIuNSwwLDAsMS0zLjYxLDAsMi41NDMsMi41NDMsMCwwLDEtLjc0My0xLjgzMiwyLjM2OSwyLjM2OSwwLDAsMSwuNzQzLTEuNzc4LDIuNjMsMi42MywwLDAsMSwzLjYxLDAsMi4zNzQsMi4zNzQsMCwwLDEsLjc0NCwxLjc3OEEyLjU0OCwyLjU0OCwwLDAsMSwzNS44LDM3LjMyOFoiLz48cG9seWdvbiBjbGFzcz0iY2xzLTIiIHBvaW50cz0iMTUuMDg2IDIyLjU4MiAxMy4yNTQgMjAuNzUgMTEuNTAyIDIyLjU4MiAxNS4wODYgMjYuMTY1IDE2LjkxNyAyNC4zMzQgMjAuNTAxIDIwLjc1IDE4LjY2OSAxOC45OTggMTUuMDg2IDIyLjU4MiIvPjxwb2x5Z29uIGNsYXNzPSJjbHMtMiIgcG9pbnRzPSIxNS4wODYgMzQuNTg2IDEzLjI1NCAzMi43NTQgMTEuNTAyIDM0LjU4NiAxNS4wODYgMzguMTcgMTYuOTE3IDM2LjMzOCAyMC41MDEgMzIuNzU0IDE4LjY2OSAzMS4wMDIgMTUuMDg2IDM0LjU4NiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTQ5LjkzNyw0OS4zODRsLTcuNjYtNy42NmExMC4xMTIsMTAuMTEyLDAsMCwwLDEuNTg4LTIuODk1LDEwLjMwOCwxMC4zMDgsMCwwLDAtLjI4LTcuNDI3LDEwLjU0OSwxMC41NDksMCwwLDAtNS41NTgtNS41NTgsMTAuMjQsMTAuMjQsMCwwLDAtOC4xMjgsMEExMC41NDksMTAuNTQ5LDAsMCwwLDI0LjM0MSwzMS40YTEwLjIzNywxMC4yMzcsMCwwLDAsMCw4LjEyN0ExMC41NDksMTAuNTQ5LDAsMCwwLDI5LjksNDUuMDg3YTkuOTg3LDkuOTg3LDAsMCwwLDQuMDY0Ljg0MSwxMC4zMjEsMTAuMzIxLDAsMCwwLDYuMjU5LTIuMDU1bDcuNjYsNy42NmExLjM2NCwxLjM2NCwwLDAsMCwyLjA1NSwwQTEuMzEsMS4zMSwwLDAsMCw0OS45MzcsNDkuMzg0Wm0tMTAuNy04LjY0MWE3LjQ0MSw3LjQ0MSwwLDAsMS0xMC41NTYsMCw3LjQ0Myw3LjQ0MywwLDAsMSwwLTEwLjU1Niw3LjQ0Myw3LjQ0MywwLDAsMSwxMC41NTYsMCw3LjQ0Myw3LjQ0MywwLDAsMSwwLDEwLjU1NloiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNSwyMy41aDlhMS41LDEuNSwwLDEsMCwwLTNIMjVhMS41LDEuNSwwLDEsMCwwLDNaIi8+PC9zdmc+",
            longDescription: "Retrieve audit logs",
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/30ece35bac024ca69de8b16bff79c413.html",
            serviceInventoryId: "SERVICE-136",
          },
          broker_id: "83d2f1bc-3a71-43ad-a8bf-133d7d2f80c5",
          catalog_id: "77f00a5c-c213-4f34-b393-13d597c7c7f0-3",
          catalog_name: "auditlog-management",
          created_at: "2020-09-04T14:00:21.635804Z",
          updated_at: "2024-04-03T15:11:23.076843Z",
        },
        {
          id: "f2117f62-6119-4f06-b4f2-1c50c7248696",
          ready: true,
          name: "auditlog-api",
          description: "[DEPRECATED] Auditlog API",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          metadata: {
            displayName: "[DEPRECATED] Audit Log Retrieval API v1",
            serviceInventoryId: "SERVICE-136",
          },
          broker_id: "0c5a2414-e7b1-4802-81e8-199f751d526f",
          catalog_id: "5b939606-3c99-11e8-b467-oiu5f89f717d-uaa",
          catalog_name: "auditlog-api",
          created_at: "2020-09-04T15:54:06.210729Z",
          updated_at: "2023-11-22T10:01:22.733926Z",
        },
        {
          id: "4a36ec67-fe2d-48f6-a544-ca931ca2cb29",
          ready: true,
          name: "content-agent",
          description:
            "Content Agent allows you to assemble the content into MTAR and export it to the transport queue.",
          bindable: true,
          instances_retrievable: true,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          metadata: {
            displayName: "Content Agent",
            longDescription:
              "Cloud Foundry based utility service that like an agent for content assembly and export to Transport Management Service or Change Management System",
            documentationUrl:
              "https://help.sap.com/viewer/p/CONTENT_AGENT_SERVICE",
            supportUrl: "https://help.sap.com/viewer/p/CONTENT_AGENT_SERVICE",
            serviceInventoryId: "SERVICE-430",
            sap: {
              tenant_aware: true,
              instance_isolation: true,
            },
          },
          broker_id: "626a2176-2dc8-49f6-8c14-3550e5595754",
          catalog_id: "ed8cd07d-3ed9-4391-a9c4-92b8ee5816b4",
          catalog_name: "content-agent",
          created_at: "2021-04-20T16:13:33.2282Z",
          updated_at: "2024-04-03T15:07:02.026834Z",
        },
        {
          id: "2345e6ef-4dd9-4a41-a6dc-850925dd1215",
          ready: true,
          name: "identity",
          description: "Cloud Identity Services",
          bindable: true,
          instances_retrievable: true,
          bindings_retrievable: true,
          plan_updateable: false,
          allow_context_updates: false,
          metadata: {
            longDescription:
              "Cloud Identity Services provide basic capabilities for user authentication.",
            documentationUrl: "https://help.sap.com/IAS",
            serviceInventoryId: "SERVICE-111",
            displayName: "Cloud Identity Services",
            dataCenterUrl: "https://eu-osb.accounts400.ondemand.com",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0ic2FwLWNsb3VkLWlkZW50aXR5LXNlcnZpY2UiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDU2IDU2Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6IzBhNmVkMTt9LmNscy0ye2ZpbGw6IzA1M2I3MDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPnNhcC1jbG91ZC1pZGVudGl0eS1zZXJ2aWNlPC90aXRsZT48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0yNi4xNTEsMzEuNmEzLjc0OCwzLjc0OCwwLDAsMC0xLjItLjgwNkEzLjY3LDMuNjcsMCwwLDAsMjMuNSwzMC41SDE5Ljc1YTMuNjQsMy42NCwwLDAsMC0xLjQ2NS4yOTNBMy43OTQsMy43OTQsMCwwLDAsMTcuMSwzMS42YTMuNzQ4LDMuNzQ4LDAsMCwwLS44MDYsMS4yQTMuNjcsMy42NywwLDAsMCwxNiwzNC4yNVYzOEgyNy4yNVYzNC4yNWEzLjY3LDMuNjcsMCwwLDAtLjI5My0xLjQ1QTMuNzQ4LDMuNzQ4LDAsMCwwLDI2LjE1MSwzMS42WiIvPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTI0LjI3NiwyOS40YTMuNzk0LDMuNzk0LDAsMCwwLC44MDYtMS4xODYsMy43NzIsMy43NzIsMCwwLDAsMC0yLjkxNSwzLjc0NSwzLjc0NSwwLDAsMC0yLjAwNy0yLjAwNywzLjc3MiwzLjc3MiwwLDAsMC0yLjkxNSwwLDMuNzk0LDMuNzk0LDAsMCwwLTEuMTg2LjgwNiwzLjc0OCwzLjc0OCwwLDAsMC0uODA2LDEuMiwzLjc3MiwzLjc3MiwwLDAsMCwwLDIuOTE1LDMuODI2LDMuODI2LDAsMCwwLDEuOTkyLDEuOTkyLDMuNzcyLDMuNzcyLDAsMCwwLDIuOTE1LDBBMy43NDgsMy43NDgsMCwwLDAsMjQuMjc2LDI5LjRaIi8+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNMzkuNzA3LDMyLjhBMy43NDUsMy43NDUsMCwwLDAsMzcuNywzMC43OTNhMy42NywzLjY3LDAsMCwwLTEuNDUtLjI5M0gzMi41YTMuNjQsMy42NCwwLDAsMC0xLjQ2NS4yOTMsMy43OTQsMy43OTQsMCwwLDAtMS4xODYuODA2LDMuNzQ4LDMuNzQ4LDAsMCwwLS44MDYsMS4yLDMuNjUyLDMuNjUyLDAsMCwwLS4yOTMsMS40NVYzOEg0MFYzNC4yNUEzLjY3LDMuNjcsMCwwLDAsMzkuNzA3LDMyLjhaIi8+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNMzcuMDI2LDI5LjRhMy43OTQsMy43OTQsMCwwLDAsLjgwNi0xLjE4NiwzLjc3MiwzLjc3MiwwLDAsMCwwLTIuOTE1LDMuNzQ1LDMuNzQ1LDAsMCwwLTIuMDA3LTIuMDA3LDMuNzcyLDMuNzcyLDAsMCwwLTIuOTE1LDAsMy43OTQsMy43OTQsMCwwLDAtMS4xODYuODA2LDMuNzQ4LDMuNzQ4LDAsMCwwLS44MDYsMS4yLDMuNzcyLDMuNzcyLDAsMCwwLDAsMi45MTUsMy44MjYsMy44MjYsMCwwLDAsMS45OTIsMS45OTIsMy43NzIsMy43NzIsMCwwLDAsMi45MTUsMEEzLjc0OCwzLjc0OCwwLDAsMCwzNy4wMjYsMjkuNFoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik00NS44NCwyMy45NjJhOC40ODksOC40ODksMCwwLDAtMTIuNzgzLTUuNzEzQTExLjU1NSwxMS41NTUsMCwwLDAsMjIuNDEsMTFDOS42MzUsMTEsMTEuMDksMjMuOTg4LDExLjA5LDIzLjk4OEExMC4yNTcsMTAuMjU3LDAsMCwwLDE0LjI4NSw0NEg0MS41YTEwLjQ4NiwxMC40ODYsMCwwLDAsNC4zNC0yMC4wMzhaTTQxLjUsNDFIMTQuMjg1YTcuMjU3LDcuMjU3LDAsMCwxLTIuMjU4LTE0LjE2MmwyLjI3OS0uNzY4LS4yMzItMi4zODljMC0uMDQyLS4zNzktNC4yMzcsMi4wMS03LjAxMywxLjM3Ny0xLjYsMy41MjQtMi41LDYuMzgxLTIuNjY2YTkuMjA5LDkuMjA5LDAsMCwxLDcuOTk0LDUuMzM5bDEuMTc2LDIuODcxLDIuNDI0LTEuMzE4QTcuNiw3LjYsMCwwLDEsMzcuNDQ5LDIwYTUuNTQ2LDUuNTQ2LDAsMCwxLDUuNDQzLDQuNTE4bC4yODgsMS41MjgsMS40MTUuNjQ2QTcuNDg2LDcuNDg2LDAsMCwxLDQxLjUsNDFaIi8+PC9zdmc+",
          },
          broker_id: "34e74589-2747-4786-bb56-72eda8366662",
          catalog_id: "8b37dc12-86d6-4ee7-a83c-fc90ba8cfa25",
          catalog_name: "identity",
          created_at: "2022-01-28T14:43:05.77551Z",
          updated_at: "2024-04-03T15:07:43.781897Z",
        },
        {
          id: "b4842a3a-df33-4cec-a879-9b4b58691845",
          ready: true,
          name: "poc-broker-test",
          description:
            "Provides an overview of any service instances and bindings that have been created by a platform.",
          bindable: true,
          instances_retrievable: true,
          bindings_retrievable: true,
          plan_updateable: true,
          allow_context_updates: false,
          tags: ["poc-broker-test"],
          metadata: {
            shareable: true,
          },
          broker_id: "a9068a87-039e-40ac-8b36-460334b3e48e",
          catalog_id: "42f3eb81-4b36-4a61-a728-7bb1c261c892",
          catalog_name: "poc-broker-test",
          created_at: "2022-02-24T14:22:07.536644Z",
          updated_at: "2022-02-24T14:22:07.642391Z",
        },
        {
          id: "7bf5d92c-c1ed-4df4-b2dd-32ff5494bfd2",
          ready: true,
          name: "print",
          description:
            "Manage print queues, connect print clients and monitor print status",
          bindable: true,
          instances_retrievable: true,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["Print", "Output Management"],
          metadata: {
            displayName: "Print Service",
            providerDisplayName: "SAP Cloud Platform",
            longDescription:
              "Manage print queues, connect print clients and monitor print status",
            createInstanceDocumentationUrl:
              "https://help.sap.com/viewer/product/SCP_PRINT_SERVICE/SHIP/en-US",
            updateInstanceDocumentationUrl:
              "https://help.sap.com/viewer/product/SCP_PRINT_SERVICE/SHIP/en-US",
            documentationURL:
              "https://help.sap.com/viewer/product/SCP_PRINT_SERVICE/SHIP/en-US",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0icHJpbnQiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDU2IDU2Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6IzA1M2I3MDt9LmNscy0ye2ZpbGw6IzBhNmVkMTt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPnByaW50PC90aXRsZT48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik01MS4xMDksMTMuODkxQTIuODc5LDIuODc5LDAsMCwwLDQ5LDEzSDQzVjdhMi44OTEsMi44OTEsMCwwLDAtLjg5MS0yLjEwOUEyLjg3OSwyLjg3OSwwLDAsMCw0MCw0SDE2YTIuODMzLDIuODMzLDAsMCwwLTIuMTU2Ljg5MUEyLjk2MiwyLjk2MiwwLDAsMCwxMyw3djZIN2EyLjgzMywyLjgzMywwLDAsMC0yLjE1Ni44OTFBMi45NjIsMi45NjIsMCwwLDAsNCwxNlYzMWEyLjg4OSwyLjg4OSwwLDAsMCwzLDNoNlYzMUg3VjE2SDQ5VjMxSDQzdjNoNmEyLjk2MiwyLjk2MiwwLDAsMCwyLjEwOS0uODQ0QTIuODQ0LDIuODQ0LDAsMCwwLDUyLDMxVjE2QTIuODkxLDIuODkxLDAsMCwwLDUxLjEwOSwxMy44OTFaTTQwLDEzSDE2VjdINDBaIi8+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNNDYsMjAuNWExLjUxMSwxLjUxMSwwLDAsMC0uNDIyLTEuMDMxQTEuMzgxLDEuMzgxLDAsMCwwLDQ0LjUsMTloLTZhMS4zNzgsMS4zNzgsMCwwLDAtMS4wNzguNDY5QTEuNTExLDEuNTExLDAsMCwwLDM3LDIwLjUsMS4zMjUsMS4zMjUsMCwwLDAsMzguNSwyMmg2QTEuMzI3LDEuMzI3LDAsMCwwLDQ2LDIwLjVaIi8+PHJlY3QgY2xhc3M9ImNscy0yIiB4PSIxOSIgeT0iMzEiIHdpZHRoPSIxOCIgaGVpZ2h0PSIzIi8+PHJlY3QgY2xhc3M9ImNscy0yIiB4PSIxOSIgeT0iMzciIHdpZHRoPSIxOCIgaGVpZ2h0PSIzIi8+PHBvbHlnb24gY2xhc3M9ImNscy0yIiBwb2ludHM9IjM3IDQzIDE5IDQzIDI4IDQ3LjEwMiAzNyA0MyIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTQyLjEwOSwyNS44OTFBMi44NzksMi44NzksMCwwLDAsNDAsMjVIMTZhMi44MzMsMi44MzMsMCwwLDAtMi4xNTYuODkxQTIuOTYyLDIuOTYyLDAsMCwwLDEzLDI4VjQ5YTIuODg5LDIuODg5LDAsMCwwLDMsM0g0MGEyLjk2MiwyLjk2MiwwLDAsMCwyLjEwOS0uODQ0QTIuODQ4LDIuODQ4LDAsMCwwLDQzLDQ5VjI4QTIuODkxLDIuODkxLDAsMCwwLDQyLjEwOSwyNS44OTFaTTQwLDQ5SDE2VjI4SDQwWiIvPjwvc3ZnPg==",
            supportURL: "https://launchpad.support.sap.com",
          },
          broker_id: "2b5b3e26-6363-4a16-abd9-930c4bcd87e7",
          catalog_id: "1e0ab901-c1b1-42e7-b4e5-82e8f409abf1",
          catalog_name: "print",
          created_at: "2022-03-10T06:17:08.046826Z",
          updated_at: "2024-04-03T15:08:51.780445Z",
        },
        {
          id: "b96b47de-0380-4aa3-95a2-da2f1e269a18",
          ready: true,
          name: "one-mds",
          description: "Service for master data integration",
          bindable: true,
          instances_retrievable: true,
          bindings_retrievable: false,
          plan_updateable: true,
          allow_context_updates: true,
          metadata: {
            displayName: "SAP Master Data Integration",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxNTAwIDE1MDAiPjxkZWZzPjxzdHlsZT4uY2xzLTF7b3BhY2l0eTowLjI7fS5jbHMtMntmaWxsOiMyMzkxYjg7fS5jbHMtM3tmaWxsOiMxZDYyYWE7fTwvc3R5bGU+PC9kZWZzPjx0aXRsZT5aZWljaGVuZmzDpGNoZSAxIEtvcGllIDY8L3RpdGxlPjxnIGlkPSJFYmVuZV8zIiBkYXRhLW5hbWU9IkViZW5lIDMiPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTY0Mi44Nyw0NzguNTEsNDYyLjY5LDg2MC43QTgxLjgxLDgxLjgxLDAsMCwxLDM1NCw5MDAuMjdoMGE4MS44MSw4MS44MSwwLDAsMS0zOS41Ny0xMDguNzJMNDk0LjU3LDQwOS4zN0E4MS44Myw4MS44MywwLDAsMSw2MDMuMywzNjkuNzloMEE4MS44Miw4MS44MiwwLDAsMSw2NDIuODcsNDc4LjUxWiIvPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTc2NS41Niw3NTAuNjMsNjMxLDEwMzQuMzdhODEuODEsODEuODEsMCwwLDEtMTA4LjcxLDM5LjU2aDBBODEuOCw4MS44LDAsMCwxLDQ4Mi43LDk2NS4yMkw2MTcuMjksNjgxLjQ4QTgxLjgsODEuOCwwLDAsMSw3MjYsNjQxLjkxaDBBODEuODIsODEuODIsMCwwLDEsNzY1LjU2LDc1MC42M1oiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMSIgY3g9IjMxMC44NyIgY3k9Ijk5OS45MSIgcj0iODEuNTMiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMSIgY3g9IjgwMi40OCIgY3k9Ijk5OS45MSIgcj0iODEuNTMiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik04MDYuNjcsNzkxLjU1YTgxLjgyLDgxLjgyLDAsMCwwLDM5LjU4LDEwOC43MmgwQTgxLjgsODEuOCwwLDAsMCw5NTUsODYwLjdsMTgwLjE5LTM4Mi4xOWE4MS44Miw4MS44MiwwLDAsMC0zOS41OC0xMDguNzJoMGE4MS44Miw4MS44MiwwLDAsMC0xMDguNzIsMzkuNThaIi8+PGNpcmNsZSBjbGFzcz0iY2xzLTEiIGN4PSIxMjk0Ljc2IiBjeT0iOTk5LjkxIiByPSI4MS41MyIvPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTEyNDguMjQsNzY1bC0xMjUsMjY5LjM0YTgxLjgxLDgxLjgxLDAsMCwxLTEwOC43MSwzOS41NmgwQTgxLjgsODEuOCwwLDAsMSw5NzUsOTY1LjIybDEyNS0yNjkuMzNhODEuNzksODEuNzksMCwwLDEsMTA4LjctMzkuNTdoMEE4MS44MSw4MS44MSwwLDAsMSwxMjQ4LjI0LDc2NVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik02MTguODYsNDc4LjUxLDQzOC42Nyw4NjAuN0E4MS44LDgxLjgsMCwwLDEsMzMwLDkwMC4yN2gwYTgxLjgyLDgxLjgyLDAsMCwxLTM5LjU4LTEwOC43Mkw0NzAuNTYsNDA5LjM3YTgxLjgyLDgxLjgyLDAsMCwxLDEwOC43Mi0zOS41OGgwQTgxLjgyLDgxLjgyLDAsMCwxLDYxOC44Niw0NzguNTFaIi8+PHBhdGggY2xhc3M9ImNscy0yIiBkPSJNNTY0LjIyLDUyMS41Niw0MzAuNDEsNTQ5Ljg0YTgxLjg0LDgxLjg0LDAsMCwxLTk4LjE1LTYxLjI5aDBhODEuODEsODEuODEsMCwwLDEsNjEuMzEtOTguMTJsMTMzLjgxLTI4LjI4YTgxLjg0LDgxLjg0LDAsMCwxLDk4LjE1LDYxLjI5aDBBODEuODEsODEuODEsMCwwLDEsNTY0LjIyLDUyMS41NloiLz48cGF0aCBjbGFzcz0iY2xzLTMiIGQ9Ik03NDEuNTUsNzUwLjYzLDYwNywxMDM0LjM3YTgxLjgsODEuOCwwLDAsMS0xMDguNywzOS41NmgwYTgxLjgsODEuOCwwLDAsMS0zOS41Ny0xMDguNzFMNTkzLjI3LDY4MS40OEE4MS44Miw4MS44MiwwLDAsMSw3MDIsNjQxLjkxaDBBODEuODIsODEuODIsMCwwLDEsNzQxLjU1LDc1MC42M1oiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjI4Ni44NSIgY3k9Ijk5OS45MSIgcj0iODEuNTMiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9Ijc3OC40NyIgY3k9Ijk5OS45MSIgcj0iODEuNTMiLz48cGF0aCBjbGFzcz0iY2xzLTMiIGQ9Ik05NjIuODQsNDA5LjM3YTgxLjgzLDgxLjgzLDAsMCwxLDEwOC43My0zOS41OGgwYTgxLjgyLDgxLjgyLDAsMCwxLDM5LjU3LDEwOC43Mkw5MzEsODYwLjdhODEuODEsODEuODEsMCwwLDEtMTA4LjczLDM5LjU3aDBhODEuODEsODEuODEsMCwwLDEtMzkuNTctMTA4LjcyIi8+PGNpcmNsZSBjbGFzcz0iY2xzLTMiIGN4PSIxMjcwLjc1IiBjeT0iOTk5LjkxIiByPSI4MS41MyIvPjxwYXRoIGNsYXNzPSJjbHMtMyIgZD0iTTEyMjQuMjIsNzY1bC0xMjUsMjY5LjM0YTgxLjgxLDgxLjgxLDAsMCwxLTEwOC43MSwzOS41NmgwQTgxLjgsODEuOCwwLDAsMSw5NTEsOTY1LjIybDEyNS0yNjkuMzNhODEuOCw4MS44LDAsMCwxLDEwOC43MS0zOS41N2gwQTgxLjgsODEuOCwwLDAsMSwxMjI0LjIyLDc2NVoiLz48L2c+PC9zdmc+",
            longDescription:
              "SAP Master Data Integration offers master data synchronization across SAP solutions and is a central access layer for data sharing and distribution. SAP Master Data Integration can only be used for SAP to SAP Integration. It must not be directly accessed for 3rd party master data integration scenarios with SAP. SAP Master Data Orchestration is part of SAP Master Data Integration and can only be used in conjunction with SAP Master Data Integration.",
            documentationUrl:
              "https://help.sap.com/viewer/product/SAP_MASTER_DATA_INTEGRATION/CLOUD/en-US",
          },
          broker_id: "77d66285-d0f4-4906-a46d-203538d8ee14",
          catalog_id: "40dc21fb-08bd-4835-8300-739ad3028970",
          catalog_name: "one-mds",
          created_at: "2022-12-16T08:30:57.81019Z",
          updated_at: "2024-04-03T15:13:26.306769Z",
        },
      ],
    });
  } else if (ns === "namespace1" && n === "secret1") {
    res.json({
      num_items: 6,
      items: [
        {
          id: "7dc306e2-c1b5-46b3-8237-bcfbda56ba66",
          ready: true,
          name: "service-manager",
          description:
            "The central registry for service brokers and platforms in SAP Business Technology Platform",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          metadata: {
            createBindingDocumentationUrl:
              "https://help.sap.com/viewer/09cc82baadc542a688176dce601398de/Cloud/en-US/1ca5bbeac19340ce959e82b51b2fde1e.html",
            discoveryCenterUrl:
              "https://discovery-center.cloud.sap/serviceCatalog/service-management",
            displayName: "Service Manager",
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/f13b6c63eef341bc8b7d25b352401c92.html",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMjI5IiBkYXRhLW5hbWU9IkxheWVyIDIyOSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIiB2aWV3Qm94PSIwIDAgNTYgNTYiPjxkZWZzPjxzdHlsZT4uY2xzLTF7ZmlsbDojMGE2ZWQxO30uY2xzLTJ7ZmlsbDojMDUzYjcwO308L3N0eWxlPjwvZGVmcz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0yOCw3YTMsMywwLDEsMS0zLDMsMywzLDAsMCwxLDMtM20wLTNhNiw2LDAsMSwwLDYsNiw2LjAwNyw2LjAwNywwLDAsMC02LTZaIi8+PHBhdGggY2xhc3M9ImNscy0xIiBkPSJNMjgsNDNhMywzLDAsMSwxLTMsMywzLDMsMCwwLDEsMy0zbTAtM2E2LDYsMCwxLDAsNiw2LDYuMDA3LDYuMDA3LDAsMCwwLTYtNloiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMywyNXY2SDdWMjVoNm0zLTNINFYzNEgxNlYyMloiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00OSwyNXY2SDQzVjI1aDZtMy0zSDQwVjM0SDUyVjIyWiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTM3LDI2LjEyNUE3LjEzMyw3LjEzMywwLDAsMSwyOS44NzUsMTlhMS4xMjUsMS4xMjUsMCwwLDEsMi4yNSwwQTQuODc5LDQuODc5LDAsMCwwLDM3LDIzLjg3NWExLjEyNSwxLjEyNSwwLDAsMSwwLDIuMjVaIi8+PHBhdGggY2xhc3M9ImNscy0yIiBkPSJNMTksMjYuMTI1YTEuMTI1LDEuMTI1LDAsMCwxLDAtMi4yNUE0Ljg3OSw0Ljg3OSwwLDAsMCwyMy44NzUsMTlhMS4xMjUsMS4xMjUsMCwwLDEsMi4yNSwwQTcuMTMzLDcuMTMzLDAsMCwxLDE5LDI2LjEyNVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNSwzOC4xMjVBMS4xMjUsMS4xMjUsMCwwLDEsMjMuODc1LDM3LDQuODgsNC44OCwwLDAsMCwxOSwzMi4xMjVhMS4xMjUsMS4xMjUsMCwwLDEsMC0yLjI1QTcuMTMzLDcuMTMzLDAsMCwxLDI2LjEyNSwzNywxLjEyNSwxLjEyNSwwLDAsMSwyNSwzOC4xMjVaIi8+PHBhdGggY2xhc3M9ImNscy0yIiBkPSJNMzEsMzguMTI1QTEuMTI1LDEuMTI1LDAsMCwxLDI5Ljg3NSwzNyw3LjEzMyw3LjEzMywwLDAsMSwzNywyOS44NzVhMS4xMjUsMS4xMjUsMCwwLDEsMCwyLjI1QTQuODgsNC44OCwwLDAsMCwzMi4xMjUsMzcsMS4xMjUsMS4xMjUsMCwwLDEsMzEsMzguMTI1WiIvPjwvc3ZnPg==",
            longDescription:
              "SAP Service Manager allows you to consume platform services in any connected runtime environment, track service instances creation, and share services and service instances between different environments.",
            serviceInventoryId: "SERVICE-324",
            shareable: true,
            supportUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/5dd739823b824b539eee47b7860a00be.html",
          },
          broker_id: "c7788cfd-3bd9-4f66-9bd8-487a23142f2c",
          catalog_id: "6e6cc910-c2f7-4b95-a725-c986bb51bad7",
          catalog_name: "service-manager",
          created_at: "2020-08-09T11:31:20.082571Z",
          updated_at: "2024-04-03T15:02:08.45919Z",
        },
        {
          id: "b3f88a98-4076-4d8b-b519-1c5222c9b178",
          ready: true,
          name: "lps-service",
          description: "Service for integrating with LPS Service",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: true,
          allow_context_updates: false,
          tags: ["lps", "service"],
          metadata: {
            shareable: false,
            displayName: "LPS Service",
          },
          broker_id: "bd821762-2eb0-407a-8b09-d80330750d1d",
          catalog_id: "72d71e2f-aa3e-4fc9-bfc0-8a5a8b541570",
          catalog_name: "lps-service",
          created_at: "2020-08-10T07:34:28.809068Z",
          updated_at: "2024-04-03T15:02:34.107748Z",
        },
        {
          id: "a5387c0b-141b-4b66-bb14-9fdb032e6eaf",
          ready: true,
          name: "saas-registry",
          description:
            "Service for application providers to register multitenant applications and services",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["SaaS"],
          metadata: {
            documentationUrl:
              "https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/5e8a2b74e4f2442b8257c850ed912f48.html",
            serviceInventoryId: "SERVICE-859",
            displayName: "SaaS Provisioning Service",
            imageUrl:
              "data:image/png;base64,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",
          },
          broker_id: "e1c79edb-21eb-4b15-b873-176fc64cc438",
          catalog_id: "lps-saas-registry-service-broker",
          catalog_name: "saas-registry",
          created_at: "2020-08-10T07:35:37.447784Z",
          updated_at: "2024-04-03T15:02:40.702428Z",
        },
        {
          id: "8627a19b-c397-4b1a-b297-6281bd46d8c3",
          ready: true,
          name: "destination",
          description:
            "Provides a secure and reliable access to destination and certificate configurations",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["destination", "conn", "connsvc"],
          metadata: {
            longDescription:
              "Use the Destination service to provide your cloud applications with access to destination and certificate configurations in a secure and reliable way",
            documentationUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/34010ace6ac84574a4ad02f5055d3597.html",
            providerDisplayName: "SAP SE",
            serviceInventoryId: "SERVICE-171",
            displayName: "Destination",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iZGVzdGluYXRpb24iIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDU2IDU2Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6IzVhN2E5NDt9LmNscy0ye2ZpbGw6IzAwOTJkMTt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPmRlc3RpbmF0aW9uPC90aXRsZT48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xOSw1MkgxMC4wOTRhMy4wNzIsMy4wNzIsMCwwLDEtMi4yLS44NDRBMi44MzcsMi44MzcsMCwwLDEsNyw0OVYxNkwxOSw0SDQwYTIuODQxLDIuODQxLDAsMCwxLDIuMTU2Ljg5MUEyLjk2MiwyLjk2MiwwLDAsMSw0Myw3djNINDBWN0gyMnY5YTIuODQ0LDIuODQ0LDAsMCwxLS44OTEsMi4xNTZBMi45NjIsMi45NjIsMCwwLDEsMTksMTlIMTBWNDloOVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNy45MzgsNDEuODYzLDI0LjcxNiw0MC4ybC0yLjAyNCwxLjg1OUwyMC4xMTUsMzkuNTJsMS43NjQtMS43NjQtMS4zNjctMy41MjdMMTgsMzQuMmwwLTMuNTc2aDIuNDc5bDEuNDctMy41NTEtMS44MzQtMS44NDUsMi41My0yLjU3NCwxLjkxMiwxLjkxMSwzLjM4MS0xLjQtLjAxNS0yLjc1NCwzLjc2NS4wMTd2Mi43MzdsMy4zOCwxLjRMMzcuMDg0LDIyLjgsMzkuNTEsMjUuNDhsLTEuNzY0LDEuNzY0LDEuNCwzLjM4MSwyLjY2Ni4xODdWMzIuNWgzVjMwLjgxMmEzLjEyNSwzLjEyNSwwLDAsMC0zLjE4OC0zLjE4N2gtLjAybC4wODItLjA3OWEzLjI3NSwzLjI3NSwwLDAsMCwuODU4LTIuMjE4LDMuMDc2LDMuMDc2LDAsMCwwLS45MTQtMi4yMjFsLTIuNDI2LTIuNDI1YTMuMjYxLDMuMjYxLDAsMCwwLTQuNDk0LDBsLS4wMjMuMDIzdi0uMDE3QTMuMTI1LDMuMTI1LDAsMCwwLDMxLjUsMTcuNUgyOC4xMjVhMy4xMjMsMy4xMjMsMCwwLDAtMy4xODcsMy4xODh2LjAxN2wtLjAyNC0uMDIzYTMuMjYxLDMuMjYxLDAsMCwwLTQuNDk0LDBsLTIuNDI2LDIuNDI1YTMuMDgsMy4wOCwwLDAsMC0uOTE0LDIuMjIxLDMuMzA5LDMuMzA5LDAsMCwwLC45MTQsMi4yNzRsLjAyNC4wMjNIMThhMy4xMjMsMy4xMjMsMCwwLDAtMy4xODcsMy4xODd2My4zNzZhMy4xNzcsMy4xNzcsMCwwLDAsLjg4NCwyLjIxNywzLjA4OCwzLjA4OCwwLDAsMCwyLjMuOTdoLjAxOGwtLjAyNC4wMjNhMy4yMiwzLjIyLDAsMCwwLDAsNC40OTVsMi40MjYsMi40MjVhMy4yNDUsMy4yNDUsMCwwLDAsNC41MTgtLjAyM3YuMDE3YTMuMTc4LDMuMTc4LDAsMCwwLC44ODQsMi4yMTgsMy4wODgsMy4wODgsMCwwLDAsMi4zLjk3aDEuNjg4di0zbC0xLjg3NS0uMTg4WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTI5LjgxMywyOS41QTIuOTU4LDIuOTU4LDAsMCwxLDMyLjM1MiwzMUgzNS42YTUuOTg3LDUuOTg3LDAsMSwwLTcuMjg2LDcuMjg3VjM1LjAzOWEyLjk1NiwyLjk1NiwwLDAsMS0xLjUtMi41MzlBMywzLDAsMCwxLDI5LjgxMywyOS41WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTQzLjg2OSw0NS4yNzhsLjI2NC0uMjY1YTQuNTE0LDQuNTE0LDAsMCwwLDAtNi4zNjVMNDAuNzgxLDM1LjNhNC41MTYsNC41MTYsMCwwLDAtNi4zNjYsMGwtLjI2NC4yNjUtMy4xNjctMy4xNjctMS41OTEsMS41OTEsMy4xNjcsMy4xNjctLjI2NS4yNjRhNC41MTYsNC41MTYsMCwwLDAsMCw2LjM2NmwzLjM1MywzLjM1MmE0LjUxNSw0LjUxNSwwLDAsMCw2LjM2NSwwbC4yNjUtLjI2NEw0Ny40MDksNTIsNDksNTAuNDA5Wk0zNC42NDEsNDMuMmwtLjctLjdhMi40LDIuNCwwLDAsMSwwLTMuMzgxbDIuMTc3LTIuMTc2YTIuNCwyLjQsMCwwLDEsMy4zOCwwbC43LjdabTcuODQ0LjExLTIuMTc3LDIuMTc2YTIuNCwyLjQsMCwwLDEtMy4zOCwwbC0uNy0uNyw1LjU1Ny01LjU1Ny43LjdBMi40LDIuNCwwLDAsMSw0Mi40ODUsNDMuMzA4WiIvPjwvc3ZnPg==",
            supportUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/e5580c5dbb5710149e53c6013301a9f2.html",
          },
          broker_id: "624a27b3-14b6-4317-a71e-5506896d0ce4",
          catalog_id: "a8683418-15f9-11e7-873e-02667c123456",
          catalog_name: "destination",
          created_at: "2020-08-10T14:58:38.756598Z",
          updated_at: "2024-04-03T15:03:15.652954Z",
        },
        {
          id: "547b140d-9dfc-469c-85b1-0680c14ce1be",
          ready: true,
          name: "connectivity",
          description:
            "Establishes a secure and reliable connectivity between cloud applications and on-premise systems.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: true,
          allow_context_updates: false,
          tags: ["connectivity", "conn", "connsvc"],
          metadata: {
            longDescription:
              "Use the Connectivity service to establish secure and reliable connectivity between your cloud applications and on-premise systems running in isolated networks.",
            documentationUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/34010ace6ac84574a4ad02f5055d3597.html",
            providerDisplayName: "SAP SE",
            serviceInventoryId: "SERVICE-169",
            displayName: "Connectivity",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0ic2FwLWhhbmEtY2xvdWQtY29ubmVjdG9yIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1NiA1NiI+PGRlZnM+PHN0eWxlPi5jbHMtMXtmaWxsOiMwMDkyZDE7fS5jbHMtMntmaWxsOiM1YTdhOTQ7fTwvc3R5bGU+PC9kZWZzPjx0aXRsZT5zYXAtaGFuYS1jbG91ZC1jb25uZWN0b3I8L3RpdGxlPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTQxLjUsNDloLTlhMS41LDEuNSwwLDAsMCwwLDNoOWExLjUsMS41LDAsMCwwLDAtM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00OC45OTEsMjVIMjUuMDA5QTMuMDA5LDMuMDA5LDAsMCwwLDIyLDI4LjAwOVY0Mi45OTFBMy4wMDksMy4wMDksMCwwLDAsMjUuMDA5LDQ2SDQ4Ljk5MUEzLjAwOSwzLjAwOSwwLDAsMCw1Miw0Mi45OTFWMjguMDA5QTMuMDA5LDMuMDA5LDAsMCwwLDQ4Ljk5MSwyNVptMCwxOEwyNSw0Mi45OTEsMjUuMDA5LDI4SDQ4Ljk5MWwuMDA5LjAwOVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0xOS4xMDksN2E2LjQ1Nyw2LjQ1NywwLDAsMSw1Ljg2NSw0LjAzNGwxLjMwNiwzLjI4OUwyOS4zMSwxMi41YTMuOTE5LDMuOTE5LDAsMCwxLDIuMDQzLS41OTEsMy45ODcsMy45ODcsMCwwLDEsMy45MTQsMy4yNDlsLjI4OCwxLjUyOSwxLjQxNS42NDVhNS4zNTEsNS4zNTEsMCwwLDEsMyw0LjY3SDQzYTguMzU2LDguMzU2LDAsMCwwLTQuNzg1LTcuNEE2Ljk0MSw2Ljk0MSwwLDAsMCwyNy43NjIsOS45MjgsOS40NDksOS40NDksMCwwLDAsMTkuMDU1LDRDOC43LDQuNTQ4LDkuOCwxNC42MjEsOS44LDE0LjYyMUE4LjM4Nyw4LjM4NywwLDAsMCwxMi40MSwzMC45ODZIMTl2LTNIMTIuNDFhNS4zODcsNS4zODcsMCwwLDEtMS42NzUtMTAuNTE1bDIuMzA4LS43NTlMMTIuNzgxLDE0LjNhOC4xMSw4LjExLDAsMCwxLDEuNS01LjI4NEE2LjUsNi41LDAsMCwxLDE5LjEwOSw3WiIvPjwvc3ZnPg==",
            supportUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/e5580c5dbb5710149e53c6013301a9f2.html",
          },
          broker_id: "e453233d-64e9-46c7-a1fd-ee164f561309",
          catalog_id: "7e2071bd-3e15-4839-8615-c6adf8d58ad0",
          catalog_name: "connectivity",
          created_at: "2020-08-10T16:46:27.305722Z",
          updated_at: "2024-04-03T15:03:20.801493Z",
        },
        {
          id: "70da63ba-36c0-4f5b-8b64-63e02e501d44",
          ready: true,
          name: "metering-service",
          description:
            "Record usage data for commercial purposes like billing, charging, and resource planning.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: true,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["metering", "reporting"],
          metadata: {
            documentationUrl:
              "https://int.controlcenter.ondemand.com/index.html#/knowledge_center/articles/879701d81a314fe59a1ae48c56ab2526",
            serviceInventoryId: "SERVICE-367",
            displayName: "Metering Service",
          },
          broker_id: "967da469-6e7b-4d6e-ba9b-e5c32ce5027d",
          catalog_id: "metering-service-broker",
          catalog_name: "metering-service",
          created_at: "2020-08-12T13:15:46.933069Z",
          updated_at: "2024-04-03T15:03:30.855055Z",
        },
      ],
    });
  } else if (ns === "namespace2" && n === "secret3") {
    res.json({
      num_items: 2,
      items: [
        {
          id: "8627a19b-c397-4b1a-b297-6281bd46d8c3",
          ready: true,
          name: "destination",
          description:
            "Provides a secure and reliable access to destination and certificate configurations",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: false,
          allow_context_updates: false,
          tags: ["destination", "conn", "connsvc"],
          metadata: {
            longDescription:
              "Use the Destination service to provide your cloud applications with access to destination and certificate configurations in a secure and reliable way",
            documentationUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/34010ace6ac84574a4ad02f5055d3597.html",
            providerDisplayName: "SAP SE",
            serviceInventoryId: "SERVICE-171",
            displayName: "Destination",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0iZGVzdGluYXRpb24iIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDU2IDU2Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6IzVhN2E5NDt9LmNscy0ye2ZpbGw6IzAwOTJkMTt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPmRlc3RpbmF0aW9uPC90aXRsZT48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xOSw1MkgxMC4wOTRhMy4wNzIsMy4wNzIsMCwwLDEtMi4yLS44NDRBMi44MzcsMi44MzcsMCwwLDEsNyw0OVYxNkwxOSw0SDQwYTIuODQxLDIuODQxLDAsMCwxLDIuMTU2Ljg5MUEyLjk2MiwyLjk2MiwwLDAsMSw0Myw3djNINDBWN0gyMnY5YTIuODQ0LDIuODQ0LDAsMCwxLS44OTEsMi4xNTZBMi45NjIsMi45NjIsMCwwLDEsMTksMTlIMTBWNDloOVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0yNy45MzgsNDEuODYzLDI0LjcxNiw0MC4ybC0yLjAyNCwxLjg1OUwyMC4xMTUsMzkuNTJsMS43NjQtMS43NjQtMS4zNjctMy41MjdMMTgsMzQuMmwwLTMuNTc2aDIuNDc5bDEuNDctMy41NTEtMS44MzQtMS44NDUsMi41My0yLjU3NCwxLjkxMiwxLjkxMSwzLjM4MS0xLjQtLjAxNS0yLjc1NCwzLjc2NS4wMTd2Mi43MzdsMy4zOCwxLjRMMzcuMDg0LDIyLjgsMzkuNTEsMjUuNDhsLTEuNzY0LDEuNzY0LDEuNCwzLjM4MSwyLjY2Ni4xODdWMzIuNWgzVjMwLjgxMmEzLjEyNSwzLjEyNSwwLDAsMC0zLjE4OC0zLjE4N2gtLjAybC4wODItLjA3OWEzLjI3NSwzLjI3NSwwLDAsMCwuODU4LTIuMjE4LDMuMDc2LDMuMDc2LDAsMCwwLS45MTQtMi4yMjFsLTIuNDI2LTIuNDI1YTMuMjYxLDMuMjYxLDAsMCwwLTQuNDk0LDBsLS4wMjMuMDIzdi0uMDE3QTMuMTI1LDMuMTI1LDAsMCwwLDMxLjUsMTcuNUgyOC4xMjVhMy4xMjMsMy4xMjMsMCwwLDAtMy4xODcsMy4xODh2LjAxN2wtLjAyNC0uMDIzYTMuMjYxLDMuMjYxLDAsMCwwLTQuNDk0LDBsLTIuNDI2LDIuNDI1YTMuMDgsMy4wOCwwLDAsMC0uOTE0LDIuMjIxLDMuMzA5LDMuMzA5LDAsMCwwLC45MTQsMi4yNzRsLjAyNC4wMjNIMThhMy4xMjMsMy4xMjMsMCwwLDAtMy4xODcsMy4xODd2My4zNzZhMy4xNzcsMy4xNzcsMCwwLDAsLjg4NCwyLjIxNywzLjA4OCwzLjA4OCwwLDAsMCwyLjMuOTdoLjAxOGwtLjAyNC4wMjNhMy4yMiwzLjIyLDAsMCwwLDAsNC40OTVsMi40MjYsMi40MjVhMy4yNDUsMy4yNDUsMCwwLDAsNC41MTgtLjAyM3YuMDE3YTMuMTc4LDMuMTc4LDAsMCwwLC44ODQsMi4yMTgsMy4wODgsMy4wODgsMCwwLDAsMi4zLjk3aDEuNjg4di0zbC0xLjg3NS0uMTg4WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTI5LjgxMywyOS41QTIuOTU4LDIuOTU4LDAsMCwxLDMyLjM1MiwzMUgzNS42YTUuOTg3LDUuOTg3LDAsMSwwLTcuMjg2LDcuMjg3VjM1LjAzOWEyLjk1NiwyLjk1NiwwLDAsMS0xLjUtMi41MzlBMywzLDAsMCwxLDI5LjgxMywyOS41WiIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTQzLjg2OSw0NS4yNzhsLjI2NC0uMjY1YTQuNTE0LDQuNTE0LDAsMCwwLDAtNi4zNjVMNDAuNzgxLDM1LjNhNC41MTYsNC41MTYsMCwwLDAtNi4zNjYsMGwtLjI2NC4yNjUtMy4xNjctMy4xNjctMS41OTEsMS41OTEsMy4xNjcsMy4xNjctLjI2NS4yNjRhNC41MTYsNC41MTYsMCwwLDAsMCw2LjM2NmwzLjM1MywzLjM1MmE0LjUxNSw0LjUxNSwwLDAsMCw2LjM2NSwwbC4yNjUtLjI2NEw0Ny40MDksNTIsNDksNTAuNDA5Wk0zNC42NDEsNDMuMmwtLjctLjdhMi40LDIuNCwwLDAsMSwwLTMuMzgxbDIuMTc3LTIuMTc2YTIuNCwyLjQsMCwwLDEsMy4zOCwwbC43LjdabTcuODQ0LjExLTIuMTc3LDIuMTc2YTIuNCwyLjQsMCwwLDEtMy4zOCwwbC0uNy0uNyw1LjU1Ny01LjU1Ny43LjdBMi40LDIuNCwwLDAsMSw0Mi40ODUsNDMuMzA4WiIvPjwvc3ZnPg==",
            supportUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/e5580c5dbb5710149e53c6013301a9f2.html",
          },
          broker_id: "624a27b3-14b6-4317-a71e-5506896d0ce4",
          catalog_id: "a8683418-15f9-11e7-873e-02667c123456",
          catalog_name: "destination",
          created_at: "2020-08-10T14:58:38.756598Z",
          updated_at: "2024-04-03T15:03:15.652954Z",
        },
        {
          id: "547b140d-9dfc-469c-85b1-0680c14ce1be",
          ready: true,
          name: "connectivity",
          description:
            "Establishes a secure and reliable connectivity between cloud applications and on-premise systems.",
          bindable: true,
          instances_retrievable: false,
          bindings_retrievable: false,
          plan_updateable: true,
          allow_context_updates: false,
          tags: ["connectivity", "conn", "connsvc"],
          metadata: {
            longDescription:
              "Use the Connectivity service to establish secure and reliable connectivity between your cloud applications and on-premise systems running in isolated networks.",
            documentationUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/34010ace6ac84574a4ad02f5055d3597.html",
            providerDisplayName: "SAP SE",
            serviceInventoryId: "SERVICE-169",
            displayName: "Connectivity",
            imageUrl:
              "data:image/svg+xml;base64,PHN2ZyBpZD0ic2FwLWhhbmEtY2xvdWQtY29ubmVjdG9yIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1NiA1NiI+PGRlZnM+PHN0eWxlPi5jbHMtMXtmaWxsOiMwMDkyZDE7fS5jbHMtMntmaWxsOiM1YTdhOTQ7fTwvc3R5bGU+PC9kZWZzPjx0aXRsZT5zYXAtaGFuYS1jbG91ZC1jb25uZWN0b3I8L3RpdGxlPjxwYXRoIGNsYXNzPSJjbHMtMSIgZD0iTTQxLjUsNDloLTlhMS41LDEuNSwwLDAsMCwwLDNoOWExLjUsMS41LDAsMCwwLDAtM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik00OC45OTEsMjVIMjUuMDA5QTMuMDA5LDMuMDA5LDAsMCwwLDIyLDI4LjAwOVY0Mi45OTFBMy4wMDksMy4wMDksMCwwLDAsMjUuMDA5LDQ2SDQ4Ljk5MUEzLjAwOSwzLjAwOSwwLDAsMCw1Miw0Mi45OTFWMjguMDA5QTMuMDA5LDMuMDA5LDAsMCwwLDQ4Ljk5MSwyNVptMCwxOEwyNSw0Mi45OTEsMjUuMDA5LDI4SDQ4Ljk5MWwuMDA5LjAwOVoiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik0xOS4xMDksN2E2LjQ1Nyw2LjQ1NywwLDAsMSw1Ljg2NSw0LjAzNGwxLjMwNiwzLjI4OUwyOS4zMSwxMi41YTMuOTE5LDMuOTE5LDAsMCwxLDIuMDQzLS41OTEsMy45ODcsMy45ODcsMCwwLDEsMy45MTQsMy4yNDlsLjI4OCwxLjUyOSwxLjQxNS42NDVhNS4zNTEsNS4zNTEsMCwwLDEsMyw0LjY3SDQzYTguMzU2LDguMzU2LDAsMCwwLTQuNzg1LTcuNEE2Ljk0MSw2Ljk0MSwwLDAsMCwyNy43NjIsOS45MjgsOS40NDksOS40NDksMCwwLDAsMTkuMDU1LDRDOC43LDQuNTQ4LDkuOCwxNC42MjEsOS44LDE0LjYyMUE4LjM4Nyw4LjM4NywwLDAsMCwxMi40MSwzMC45ODZIMTl2LTNIMTIuNDFhNS4zODcsNS4zODcsMCwwLDEtMS42NzUtMTAuNTE1bDIuMzA4LS43NTlMMTIuNzgxLDE0LjNhOC4xMSw4LjExLDAsMCwxLDEuNS01LjI4NEE2LjUsNi41LDAsMCwxLDE5LjEwOSw3WiIvPjwvc3ZnPg==",
            supportUrl:
              "https://help.sap.com/viewer/cca91383641e40ffbe03bdc78f00f681/Cloud/en-US/e5580c5dbb5710149e53c6013301a9f2.html",
          },
          broker_id: "e453233d-64e9-46c7-a1fd-ee164f561309",
          catalog_id: "7e2071bd-3e15-4839-8615-c6adf8d58ad0",
          catalog_name: "connectivity",
          created_at: "2020-08-10T16:46:27.305722Z",
          updated_at: "2024-04-03T15:03:20.801493Z",
        },
      ],
    });
  } else {
    res.json({
      num_items: 0,
      items: [],
    });
  }
});

app.get("/api/list-secrets", (req, res) => {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.json({
    items: [
      { namespace: "kymasystem", name: "defaultsecret" },
      { namespace: "namespace1", name: "secret1" },
      { namespace: "namespace2", name: "secret3" },
    ],
  });
});

app.get("/api/list-service-instances", (req, res) => {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.json({
    items: [
      {
        id: "service-instance-id-1",
        name: "service-instance-name-1",
        context: ["SM", "Kyma"],
        namespace: "namespace1",
        service_bindings: [
          {
            id: "service-binding-id-3",
            name: "service-binding-name-3",
            namespace: "namespace3",
          },
          {
            id: "service-binding-id-3",
            name: "service-binding-name-3",
            namespace: "namespace3",
          },
        ],
      },
      {
        id: "service-instance-id-2",
        name: "service-instance-name-2",
        context: ["SM"],
        namespace: "namespace2",
        service_bindings: [
          {
            id: "service-binding-id-2",
            name: "service-binding-name-2",
            namespace: "namespace2",
          },
        ],
      },
      {
        id: "service-instance-id-3",
        name: "service-instance-name-3",
        context: ["Kyma"],
        namespace: "namespace3",
        service_bindings: [
          {
            id: "service-binding-id-3",
            name: "service-binding-name-3",
            namespace: "namespace3",
          },
          {
            id: "service-binding-id-3",
            name: "service-binding-name-3",
            namespace: "namespace3",
          },
          {
            id: "service-binding-id-3",
            name: "service-binding-name-3",
            namespace: "namespace3",
          },
          {
            id: "service-binding-id-3",
            name: "service-binding-name-3",
            namespace: "namespace3",
          },
        ],
      },
    ],
  });
});

app.listen(port, () => {
  console.log(`Example app listening on port ${port}`);
});
