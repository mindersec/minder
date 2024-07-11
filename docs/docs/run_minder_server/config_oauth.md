---
title: Create a GitHub OAuth Application 
sidebar_position: 120
---


## Prerequisites

- [GitHub](https://github.com) account

## Steps


A legacy method for allowing users to enroll into Minder is using a GitHub OAuth application. T

1. Navigate to [GitHub Developer Settings](https://github.com/settings/profile)
2. Select "Developer Settings" from the left hand menu
3. Select "OAuth Apps" from the left hand menu
4. Select "New OAuth App"
5. Enter the following details:
   - Application Name: `Minder` (or any other name you like)
   - Homepage URL: `http://localhost:8080`
   - Authorization callback URL: `http://localhost:8080/api/v1/auth/callback/github`
   - If you are prompted to enter a `Webhook URL`, deselect the `Active` option in the `Webhook` section.
6. Select "Register Application"
7. Generate a client secret
7. Copy the "Client ID" , "Client Secret" and "Authorization callback URL" values
   into your `./server-config.yaml` file, under the `github` section.

![github oauth2 page](./images/minder-server-oauth.png)