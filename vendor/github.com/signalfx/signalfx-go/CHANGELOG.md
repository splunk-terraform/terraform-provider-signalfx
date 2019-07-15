# 1.1.0, in progress

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

# 1.0.0, in progress

Tagged!

## Added

## Updated

## Bugfixes

## Removed
