package dynamodb

import (
	"context"
	"errors"
	"github.com/aaronland/go-aws-session"
	"github.com/aaronland/go-mailinglist/database"
	"github.com/aaronland/go-mailinglist/subscription"
	aws "github.com/aws/aws-sdk-go/aws"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	aws_dynamodb "github.com/aws/aws-sdk-go/service/dynamodb"
	aws_dynamodbattribute "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	_ "log"
)

const SUBSCRIPTIONS_DEFAULT_TABLENAME string = "subscriptions"

type DynamoDBSubscriptionsDatabaseOptions struct {
	TableName   string
	BillingMode string
	CreateTable bool
}

func DefaultDynamoDBSubscriptionsDatabaseOptions() *DynamoDBSubscriptionsDatabaseOptions {

	opts := DynamoDBSubscriptionsDatabaseOptions{
		TableName:   SUBSCRIPTIONS_DEFAULT_TABLENAME,
		BillingMode: "PAY_PER_REQUEST",
		CreateTable: false,
	}

	return &opts
}

type DynamoDBSubscriptionsDatabase struct {
	database.SubscriptionsDatabase
	client  *aws_dynamodb.DynamoDB
	options *DynamoDBSubscriptionsDatabaseOptions
}

func NewDynamoDBSubscriptionsDatabaseWithDSN(dsn string, opts *DynamoDBSubscriptionsDatabaseOptions) (database.SubscriptionsDatabase, error) {

	sess, err := session.NewSessionWithDSN(dsn)

	if err != nil {
		return nil, err
	}

	return NewDynamoDBSubscriptionsDatabaseWithSession(sess, opts)
}

func NewDynamoDBSubscriptionsDatabaseWithSession(sess *aws_session.Session, opts *DynamoDBSubscriptionsDatabaseOptions) (database.SubscriptionsDatabase, error) {

	client := aws_dynamodb.New(sess)

	if opts.CreateTable {
		_, err := CreateSubscriptionsTable(client, opts)

		if err != nil {
			return nil, err
		}
	}

	db := DynamoDBSubscriptionsDatabase{
		client:  client,
		options: opts,
	}

	return &db, nil
}

func (db *DynamoDBSubscriptionsDatabase) GetSubscriptionWithAddress(addr string) (*subscription.Subscription, error) {

	req := &aws_dynamodb.GetItemInput{
		TableName: aws.String(db.options.TableName),
		Key: map[string]*aws_dynamodb.AttributeValue{
			"address": {
				S: aws.String(addr),
			},
		},
	}

	rsp, err := db.client.GetItem(req)

	if err != nil {
		return nil, err
	}

	return itemToSubscription(rsp.Item)
}

func (db *DynamoDBSubscriptionsDatabase) AddSubscription(sub *subscription.Subscription) error {

	existing_sub, err := db.GetSubscriptionWithAddress(sub.Address)

	if err != nil && !database.IsNotExist(err) {
		return err
	}

	if existing_sub != nil {
		return errors.New("Subscription already exists")
	}

	return putSubscription(db.client, db.options, sub)
}

func (db *DynamoDBSubscriptionsDatabase) RemoveSubscription(sub *subscription.Subscription) error {

	req := &aws_dynamodb.DeleteItemInput{
		TableName: aws.String(db.options.TableName),
		Key: map[string]*aws_dynamodb.AttributeValue{
			"address": {
				S: aws.String(sub.Address),
			},
		},
	}

	_, err := db.client.DeleteItem(req)

	if err != nil {
		return err
	}

	return nil
}

func (db *DynamoDBSubscriptionsDatabase) UpdateSubscription(sub *subscription.Subscription) error {

	return putSubscription(db.client, db.options, sub)
}

func (db *DynamoDBSubscriptionsDatabase) ListSubscriptionsConfirmed(ctx context.Context, callback database.ListSubscriptionsFunc) error {

	// https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Scan.html#Scan.FilterExpression

	req := &aws_dynamodb.ScanInput{
		// 	FilterExpression:     aws.String("NOT confirmed = 0"),
		// 	ProjectionExpression: aws.String("#ST, #AT"),
		TableName: aws.String(db.options.TableName),
	}

	return scanSubscriptions(ctx, db.client, req, callback)
}

func (db *DynamoDBSubscriptionsDatabase) ListSubscriptionsUnconfirmed(ctx context.Context, callback database.ListSubscriptionsFunc) error {

	req := &aws_dynamodb.ScanInput{
		// 	FilterExpression:     aws.String("NOT confirmed = 0"),
		// 	ProjectionExpression: aws.String("#ST, #AT"),
		TableName: aws.String(db.options.TableName),
	}

	return scanSubscriptions(ctx, db.client, req, callback)
}

func putSubscription(client *aws_dynamodb.DynamoDB, opts *DynamoDBSubscriptionsDatabaseOptions, sub *subscription.Subscription) error {

	item, err := aws_dynamodbattribute.MarshalMap(sub)

	if err != nil {
		return err
	}

	req := &aws_dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(opts.TableName),
	}

	_, err = client.PutItem(req)

	if err != nil {
		return err
	}

	return nil
}

func itemToSubscription(item map[string]*aws_dynamodb.AttributeValue) (*subscription.Subscription, error) {

	var sub *subscription.Subscription

	err := aws_dynamodbattribute.UnmarshalMap(item, &sub)

	if err != nil {
		return nil, err
	}

	if sub.Address == "" {
		return nil, new(database.NoRecordError)
	}

	return sub, nil
}

func scanSubscriptions(ctx context.Context, client *aws_dynamodb.DynamoDB, req *aws_dynamodb.ScanInput, callback database.ListSubscriptionsFunc) error {

	for {

		rsp, err := client.Scan(req)

		if err != nil {
			return err
		}

		for _, item := range rsp.Items {

			sub, err := itemToSubscription(item)

			if err != nil {
				return err
			}

			err = callback(sub)

			if err != nil {
				return err
			}
		}

		req.ExclusiveStartKey = rsp.LastEvaluatedKey

		if rsp.LastEvaluatedKey == nil {
			break
		}
	}

	return nil
}
