package main

import (
	"flag"
	"github.com/aaronland/go-mailinglist-database-dynamodb"
	"log"
)

func main() {

	subs_table := flag.String("subscriptions-table", dynamodb.SUBSCRIPTIONS_DEFAULT_TABLENAME, "...")
	conf_table := flag.String("confirmations-table", dynamodb.CONFIRMATIONS_DEFAULT_TABLENAME, "...")
	dsn := flag.String("dsn", "", "...")

	// table names here... or in dsn

	flag.Parse()

	subscribe_opts := dynamodb.DefaultDynamoDBSubscriptionsDatabaseOptions()
	confirm_opts := dynamodb.DefaultDynamoDBConfirmationsDatabaseOptions()

	subscribe_opts.TableName = *subs_table
	subscribe_opts.CreateTable = true

	confirm_opts.TableName = *conf_table
	confirm_opts.CreateTable = true

	var err error

	_, err = dynamodb.NewDynamoDBSubscriptionsDatabaseWithDSN(*dsn, subscribe_opts)

	if err != nil {
		log.Printf("Failed to set up %s table, %s\n", subscribe_opts.TableName, err)
	}

	_, err = dynamodb.NewDynamoDBConfirmationsDatabaseWithDSN(*dsn, confirm_opts)

	if err != nil {
		log.Printf("Failed to set up %s table, %s\n", confirm_opts.TableName, err)
	}

}
