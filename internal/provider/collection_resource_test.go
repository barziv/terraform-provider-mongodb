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

func TestAccCollectionResource(t *testing.T) {
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
				Config: testAccCollectionResourceConfig(testDB, "test_collection", ""),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("mongodb_collection.test", "database", testDB),
					tfresource.TestCheckResourceAttr("mongodb_collection.test", "name", "test_collection"),
					tfresource.TestCheckResourceAttr("mongodb_collection.test", "id", fmt.Sprintf("%s.test_collection", testDB)),
				),
			},
			// Create with validation testing
			{
				Config: testAccCollectionResourceConfig(testDB, "test_collection_with_validation", `validation {
					validator = jsonencode({
						"$jsonSchema" = {
							bsonType = "object"
							required = ["name", "age"]
							properties = {
								name = { bsonType = "string" }
								age  = { bsonType = "int" }
							}
						}
					})
				}`),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("mongodb_collection.test", "database", testDB),
					tfresource.TestCheckResourceAttr("mongodb_collection.test", "name", "test_collection_with_validation"),
					tfresource.TestCheckResourceAttr("mongodb_collection.test", "id", fmt.Sprintf("%s.test_collection_with_validation", testDB)),
				),
			},
			// ImportState testing
			{
				ResourceName:      "mongodb_collection.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s.test_collection_with_validation", testDB),
				ImportStateVerify: true,
			},
			// Update testing - should fail as updates are not supported
			{
				Config:      testAccCollectionResourceConfig(testDB, "test_collection_updated", ""),
				ExpectError: regexp.MustCompile("Updates not supported"),
			},
		},
	})
}

func testAccCollectionResourceConfig(database, name, validation string) string {
	return fmt.Sprintf(`
resource "mongodb_collection" "test" {
	database = %q
	name     = %q
	%s
}
`, database, name, validation)
}

func TestCollectionResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := resource.SchemaRequest{}
	schemaResponse := &resource.SchemaResponse{}

	NewCollectionResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Schema.Attributes == nil {
		t.Fatal("Schema attributes should not be empty")
	}
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema should not have errors: %v", schemaResponse.Diagnostics.Errors())
	}

	// Test required attributes
	databaseAttr, ok := schemaResponse.Schema.Attributes["database"].(schema.StringAttribute)
	if !ok {
		t.Fatal("database attribute should be a StringAttribute")
	}
	if !databaseAttr.Required {
		t.Error("database attribute should be required")
	}
	if databaseAttr.Description != "Name of the database where to create the collection." {
		t.Error("database attribute has incorrect description")
	}

	nameAttr, ok := schemaResponse.Schema.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("name attribute should be a StringAttribute")
	}
	if !nameAttr.Required {
		t.Error("name attribute should be required")
	}
	if nameAttr.Description != "Name of the collection to create." {
		t.Error("name attribute has incorrect description")
	}

	// Test optional attributes
	validationAttr, ok := schemaResponse.Schema.Attributes["validation"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("validation attribute should be a SingleNestedAttribute")
	}
	if validationAttr.Required {
		t.Error("validation attribute should not be required")
	}
	if validationAttr.Description != "Collection validation rules." {
		t.Error("validation attribute has incorrect description")
	}
}

func TestCollectionResourceModel(t *testing.T) {
	t.Parallel()

	model := &collectionResourceModel{
		Database: "test_db",
		Name:     "test_collection",
		Validation: &validation{
			Validator: `{"$jsonSchema":{"bsonType":"object"}}`,
		},
		Id: types.StringValue("test_db.test_collection"),
	}

	if model.Database != "test_db" {
		t.Errorf("Expected database to be 'test_db', got '%s'", model.Database)
	}
	if model.Name != "test_collection" {
		t.Errorf("Expected name to be 'test_collection', got '%s'", model.Name)
	}
	if model.Validation == nil {
		t.Error("Validation should not be nil")
	}
	if model.Validation.Validator != `{"$jsonSchema":{"bsonType":"object"}}` {
		t.Errorf("Expected validator to be '%s', got '%s'", `{"$jsonSchema":{"bsonType":"object"}}`, model.Validation.Validator)
	}
	if model.Id.ValueString() != "test_db.test_collection" {
		t.Errorf("Expected id to be 'test_db.test_collection', got '%s'", model.Id.ValueString())
	}
}

func TestParseCollectionId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		want    *collectionId
		wantErr bool
	}{
		{
			name: "valid id",
			id:   "db.collection",
			want: &collectionId{
				database:   "db",
				collection: "collection",
			},
			wantErr: false,
		},
		{
			name:    "invalid format - no dot",
			id:      "dbcollection",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			id:      "db.collection.extra",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCollectionId(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if got != nil {
					t.Error("Expected nil result but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got == nil {
					t.Error("Expected non-nil result but got nil")
				}
				if got.database != tt.want.database {
					t.Errorf("Expected database '%s', got '%s'", tt.want.database, got.database)
				}
				if got.collection != tt.want.collection {
					t.Errorf("Expected collection '%s', got '%s'", tt.want.collection, got.collection)
				}
			}
		})
	}
}
