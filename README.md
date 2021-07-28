# Meroxicam

Meroxicam is an example project that uses Meroxa to set up a security camera which posts messages to Slack when a face
is recognized by the camera.

## Quick start

Meroxicam needs [OpenCV 4](https://opencv.org/) installed, you can do this on MacOS by running `brew install opencv`.

Build the project with `make build` and run it with:
```
meroxa create endpoint grpc-endpoint --protocol grpc --stream resource-2-499379-public.accounts
USER=$(meroxa list endpoints --json | jq -r '.[] | select(.name == "grpc-endpoint") | .basic_auth_username')
PASS=$(meroxa list endpoints --json | jq -r '.[] | select(.name == "grpc-endpoint") | .basic_auth_password')
STREAM=$(meroxa list endpoints --json | jq -r '.[] | select(.name == "grpc-endpoint") | .stream')
./meroxicam -meroxa.username=$USER -meroxa.password=$PASS -meroxa.stream=$STREAM
```

## Ideas for the future

Once Meroxa provides functions Meroxicam could be adjusted to run on a low-power IoT device which periodically sends
images to a Meroxa stream where a function executes the face recognition.