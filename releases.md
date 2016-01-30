#### Releases

This document lays out the `elos` command release schedule.

Each week, on Saturday by 12pm, the `elos` command will be re-released, even if that means only updating the version.

v0.3 changelog
 - add `elos todo tag -r` which allows you to remove a tag from a task
 - Ammend confusion regarding time zones, UTC was preserved across local and server,
    however the UTC times returned from the server needed to be turned back into local time,
    and printed as such
 - remove current time prompt
