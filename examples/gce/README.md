# About gce

This example shows how to use the firebase package with Google Compute Service
credentials, using Google's standard [oauth2/google](https://godoc.org/golang.org/x/oauth2/google)
auth token providers.

## Projects and Scopes

Please note that when using `google.ComputeTokenSource`, you must create
an instance that has the appropriate scopes for Firebase's v3+ API, either by
creating a custom service account credential in the [IAM console](https://console.cloud.google.com/iam-admin/serviceaccounts)
or when creating the compute instance.

The scopes needed for Firebase v3+ are:

* https://www.googleapis.com/auth/userinfo.email
* https://www.googleapis.com/auth/firebase.database

Scopes can be passed via the `gcloud` cli tool using the `--scopes` option:

```sh
# this uses the short-hand notation available for the userinfo.email scope
$ gcloud compute instances create <INSTANCE_NAME> --scopes userinfo-email,https://www.googleapis.com/auth/firebase.database
```

You can see what scopes are assigned to an instance using this command:
```sh
$ gcloud compute instances describe <INSTANCE_NAME>
```
