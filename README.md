# Meroxicam

Meroxicam is an example project that uses Meroxa to set up a security camera which posts messages to Slack when a face
is recognized by the camera.

## Quick start

Meroxicam needs [OpenCV 4](https://opencv.org/) installed, you can do this on MacOS by running `brew install opencv`.

## Ideas for the future

Once Meroxa provides functions Meroxicam could be adjusted to run on a low-power IoT device which periodically sends
images to a Meroxa stream where a function executes the face recognition.