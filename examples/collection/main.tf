terraform {
  required_providers {
    mongodb = {
      source = "hashicorp.com/edu/mongodb"
    }
  }
}

provider "mongodb" {
  # the env variable MONGODB_URL can be used instead
  url = "mongodb://localhost:27017"
}


resource "mongodb_database" "db" {
  name = "some-database-name"
}

resource "mongodb_collection" "collection" {
  database = mongodb_database.db.name
  name     = "some-collection-name"
}

resource "mongodb_collection" "collection" {
  database = "exists-database-name"
  name     = "some-collection-name2"
}
