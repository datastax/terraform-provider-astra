## 1.0.14
[Feature] Add multi-region support

BACKWARDS INCOMPATIBILITIES / NOTES:

The Multi-Region support feature has changed the Terraform resource structure for Databases.
Specifically, the "region" field is now an array of strings, as opposed to a single string.
When updating the plugin, from a version prior to 1.0.14, to version 1.0.14 or newer, you
will have to perform a manual Terraform state migration in order to keep existing databases
under Terraform plugin management.

For each Datatbase under Terraform management:
1. Obtain the database id (ex. b3107622-429d-45ab-a6da-0252cb091c86)
2. Obtain the terraform resource name for the database (ex. "my_db", from the resouirce line in your terraform .tf file)
3. Remove the databse from the terrafrom state
```sh
   terrafrom state rm astra_database.<resource name from #2>
```
4. Edit your terrafrom resource file and convert the "region" field value from a string to an array
```sh
   region = "us-east1"
```
to
```sh
   region = ["us-east1"]
```
5. Import the database back into the Terraform state
```sh
   terraform import astra_database.<resource name from #2> <database uuid from #1>
```