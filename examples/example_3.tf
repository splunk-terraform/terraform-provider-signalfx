provider "signalfx" {
  # Other configured values
  feature_preview = {
    "feature-01": true,  // True means that the feature is enabled
    "feature-02": false, // False means that the feature is explicitly disabled
  }
}
