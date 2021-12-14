## 2.0.0 (Unreleased)

BACKWARDS INCOMPATIBILITIES / NOTES:

* Data source and Resource `database` attribute `region` has been renamed to `regions`. The value is changed from a string to a list of strings. See https://github.com/datastax/terraform-provider-astra/blob/main/RELEASE_NOTES.md for more details.

FEATURES:

* resource/database: Add Multi-Region support [GH-60] (see BACKWARDS INCOMPATIBILITIES notes above)

IMPROVEMENTS:

* resource/database: Not all database attributes are available [GH-61]

BUG FIXES:

* resource/keyspace: Incomplete resource provisioning on Astra [GH-62]

## 1.0.13 (November 23, 2021)

BUG FIXES:

* resource/access_list: Fix access-list address adds and deletes [GH-54]

## 1.0.12 (November 19, 2021)

BUG FIXES:

* resource/access_list: Access list fix [GH-49]
