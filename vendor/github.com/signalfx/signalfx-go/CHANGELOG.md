# 1.6.9, Pending

## Added

## Updated

## Bugfixes

## Removed

# 1.6.8, Pending

## Added

* Methods for Alert Muting Rules

# 1.6.7, 2019-11-05

## Bugfixes

* Added `AuthorizedWriters` to Detector model

# 1.6.6, 2019-11-05

## Bugfixes

* Detector and DashboardGroup structs modified to use a pointer for `AuthorizedWriters`.

# 1.6.5, 2019-10-30

## Added

* Additional reconnect delays upon SignalFlow socket errors to reduce load on
backend.
* Added `*JiraIntegration` methods
* Added `notification.JiraNotification`

# 1.6.4, 2019-09-27

## Added

Event Overlays now support a detector id.

# 1.6.3, 2019-09-19

## Bugfixes

* Changed detector's time fields to be `*int64`

# 1.6.2, 2019-09-16

## Added

* VictorOps integration functions

## Updated

* Adjusted `EventPublishLabelOptions.PalleteIndex` to an `*int32` to match other uses.
* SignalFlow computation Handle() method wait for handle to come in until
  returning (with timeout).
* Renamed `BinaryPayload` to `DataPayload` in the `messages` package.
* Exported `BinaryMessageHeader` and `DataMessageHeader` from `messages`
  package to facilitate low-level SignalFlow parsing.

## Bugfixes

* SignalFlow client connection handling was refactored to prevent deadlocks
  that could occur on reconnects and bad authentication.

## Removed

# 1.6.1, 2019-08-16

## Updated

* Adjusted detector.CreateUpdateDetectorRequest to use pointer for Rules

# 1.6.0, 2019-08-16

## Added

* Added `*GCPIntegration` methods
* Added `*Opsgenie` methods
* Added `*PagerDutyIntegration` methods
* Added `*SlackIntegration` methods

## Updated

* `Detector.Rules` now uses `Notification` as it's type instead of an untyped `[]map[string]interface{}`.

## Removed
* Renamed `integration.GcpIntegration` and it's sub-types to `GCP`, fixing case.

# 1.5.0, 2019-08-05

## Added

* Add OrgToken methods

## Bugfixes

* Properly recognize the SignalFlow keep alive event message and ignore it.

## Updated

* Moved various notification bits into a `notification` package

# 1.4.0, 2019-07-29

## Added

* Add `*AzureIntregration` functions to client.

## Updated

## Bugfixes

## Removed

# 1.3.0, 2019-07-24

## Added

* Added OpenAPI code for integrations, experimental for now.
* Add `*AwsCloudWatchIntegration` functions to client.

## Removed

* Removed `credentialName` from Opsgenie notifications, not a real field in the API.

# 1.2.0, 2019-07-16

## Updated

* Many numeric properties have been adjusted to pointers to play better with Go's JSON (un)marshaling.

# 1.1.0, 2019-07-15

## Added
* Added `DashboardConfigs` to `CreateUpdateDashboardRequest`
* `DashboardGroupCreate` now has an option to create an empty group.

## Updated
* Many types have been changed to pointers to add (de)serialization
* Moved `StringOrSlice` into a `util` package, cuz all projects must have one

## Bugfixes
* Switched to `StringOrSlice` for some fields that needed it.
* Added `StringOrInteger` to handle failures in some Chart filter responses, thanks to (doctornkz)[https://github.com/doctornkz] for flagging!

## Removed

# 1.0.0, Forgot the date

Tagged!

## Added

## Updated

## Bugfixes

## Removed
