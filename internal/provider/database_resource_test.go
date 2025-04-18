package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestAccDatabaseResource(t *testing.T) {
	// Skip if not running acceptance tests
	if testing.Short() {
		t.Skip("Skipping acceptance test")
	}

	// Setup MongoDB test client
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Clean up test database before and after tests
	testDB := "terraform_test_db"
	err = client.Database(testDB).Drop(context.Background())
	if err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}
	defer client.Database(testDB).Drop(context.Background())

	tfresource.Test(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Create and Read testing
			{
				Config: testAccDatabaseResourceConfig(testDB),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("mongodb_database.test", "name", testDB),
					tfresource.TestCheckResourceAttr("mongodb_database.test", "id", testDB),
				),
			},
			// ImportState testing
			{
				ResourceName:      "mongodb_database.test",
				ImportState:       true,
				ImportStateId:     testDB,
				ImportStateVerify: true,
			},
			// Update testing - should fail as updates are not supported
			{
				Config:      testAccDatabaseResourceConfig("updated_" + testDB),
				ExpectError: regexp.MustCompile("Updates not supported"),
			},
		},
	})
}

func testAccDatabaseResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "mongodb_database" "test" {
	name = %q
}
`, name)
}

func TestDatabaseResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := resource.SchemaRequest{}
	schemaResponse := &resource.SchemaResponse{}

	NewDatabaseResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Schema.Attributes == nil {
		t.Fatal("Schema attributes should not be empty")
	}
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema should not have errors: %v", schemaResponse.Diagnostics.Errors())
	}

	// Test required attributes
	nameAttr, ok := schemaResponse.Schema.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("name attribute should be a StringAttribute")
	}
	if !nameAttr.Required {
		t.Error("name attribute should be required")
	}
	if nameAttr.Description != "Name of the database to create." {
		t.Error("name attribute has incorrect description")
	}
}

func TestDatabaseResourceModel(t *testing.T) {
	t.Parallel()

	model := &databaseResourceModel{
		Name: "test_db",
		Id:   types.StringValue("test_db"),
	}

	if model.Name != "test_db" {
		t.Errorf("Expected name to be 'test_db', got '%s'", model.Name)
	}
	if model.Id.ValueString() != "test_db" {
		t.Errorf("Expected id to be 'test_db', got '%s'", model.Id.ValueString())
	}
}
