## Version 2.0.0 Notes

### Database Resource Changes

The Multi-Region support feature has changed the Terraform resource structure for Databases.
Specifically, the "region" field has been renamed to "regions", and is now an array of strings,
as opposed to a single string. When updating the plugin from 1.x to 2.x, you will have to
perform a manual Terraform state migration in order to keep existing databases under Terraform
plugin management.

Please follow the below steps to migrate your configurations:

For each Datatbase under Terraform management:
1. Obtain the database id (ex. b3107622-429d-45ab-a6da-0252cb091c86)
2. Obtain the Terraform resource name for the database (ex. "my_db", from the resource line in your Terraform .tf file)
3. Remove the database from the Terraform state
```sh
   terraform state rm astra_database.my_db
```
4. Edit your Terraform resource file and convert the "region" field to "regions, and the value from a string to an array
```sh
   region = "us-east1"
```
to
```sh
   regions = ["us-east1"]
```
5. Import the database back into the Terraform state
```sh
   terraform import astra_database.my_db b3107622-429d-45ab-a6da-0252cb091c86
```
6. To verify, execute
```sh
   terraform show
```
which should show that the deployed "region" is now a "regions" attribute with a list of a single string.

### Database Data Source changes

If you define any external Astra Database Data Sources, you will need to update the definitions for them
as well. The same change made in the Resource schema has been made in the Data Source schema. However, for
Data Sources, you only need to remove the definition, apply, then re-add the definition and apply again.

1. Comment out (or remove) any "data" definitions for Astra databases in your Terraform files.
```sh
   #data "astra_database" "my_ext_db" {
   #  database_id = "b3107622-429d-45ab-a6da-0252cb091c86"
   #}
```
2. Apply the change
```sh
   terraform apply
```
3. Re-add the "data" definition.
```sh
   data "astra_database" "my_ext_db" {
     database_id = "b3107622-429d-45ab-a6da-0252cb091c86"
  }
```
4. Apply the change
```sh
   terraform apply
```


### Multi-Region Notes

As of version 2.0.0, the Astra provider now supports deploying to multiple regions. This can be done in a
single Terraform apply (with all regions specified in the "regions" array when creating your Astra database),
or with an incremental approach (by creating your database with 1 region in the array and then adding new
regions one by one). However, there are a few caveats:

#### Terminating a Database with Multiple regions
Currently, there is a bug in Astra that doesn't allow for a database to be terminated if the database has
more than one datacenter in multiple regions. If you try to remove a database that has multiple regions, it
may get stuck in the MAINTENANCE or TERMINATING states. To avoid this, you should apply changes to your
database so that it only has a single region and is ACTIVE before attempting to terminate the database.

#### Importance of the First Region
The first region defined in your database "regions" definition will be the region that the database is
initially created in. While you can add multiple regions, you can NOT remove the initial region, even if
your database is successfully deployed to another region. If you no longer want your database to be deployed
to this initial region, you must delete the database and recreate it in your other desired region(s). This
is a limitation of Astra currently, and the Terraform provider does not prevent you from trying to do this.

#### Adding and Terminating regions
The provider allows adding and/or terminating multiple regions in a single Terraform apply action. However,
the implementation is such that all regions to be added are handled before any regions to be terminated are
handled. Additionally, all adds and terminates are done one at a time (current Astra restriction). Each of
these actions can take some time, so it is not recommended to attempt to add and delete many regions at the
same time.
