package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBAdapter struct {
	client       *mongo.Client
	database     *mongo.Database
	databaseName string
}

func NewMongoDBAdapter(ctx context.Context, connectionString, databaseName string) (*MongoDBAdapter, error) {
	// Extract database name from connection string if not provided
	dbName := databaseName
	if dbName == "" {
		dbName = extractMongoDBNameFromConnStr(connectionString)
	}

	clientOpts := options.Client().
		ApplyURI(connectionString).
		SetMaxPoolSize(5).
		SetMinPoolSize(1).
		SetMaxConnIdleTime(5 * time.Minute)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDBAdapter{
		client:       client,
		database:     client.Database(dbName),
		databaseName: dbName,
	}, nil
}

func extractMongoDBNameFromConnStr(connStr string) string {
	if idx := strings.Index(connStr, "?"); idx != -1 {
		connStr = connStr[:idx]
	}
	parts := strings.Split(connStr, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "test"
}

func (m *MongoDBAdapter) CreateIndex(ctx context.Context, params IndexParams) error {
	exists, err := m.IndexExists(ctx, params.IndexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		return ErrIndexAlreadyExists
	}

	collection := m.database.Collection(params.TableName)

	// Build index keys
	keys := bson.D{}
	for _, col := range params.ColumnNames {
		keys = append(keys, bson.E{Key: col, Value: 1})
	}

	indexModel := mongo.IndexModel{
		Keys: keys,
		Options: options.Index().
			SetName(params.IndexName).
			SetUnique(params.Unique).
			SetBackground(true), // Non-blocking index creation
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func (m *MongoDBAdapter) DropIndex(ctx context.Context, indexName string) error {
	// Find which collection has this index
	collName, err := m.findCollectionForIndex(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to find collection for index: %w", err)
	}

	if collName == "" {
		// Index doesn't exist, nothing to do
		return nil
	}

	collection := m.database.Collection(collName)
	_, err = collection.Indexes().DropOne(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	return nil
}

func (m *MongoDBAdapter) findCollectionForIndex(ctx context.Context, indexName string) (string, error) {
	collections, err := m.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return "", err
	}

	for _, collName := range collections {
		if strings.HasPrefix(collName, "system.") {
			continue
		}

		cursor, err := m.database.Collection(collName).Indexes().List(ctx)
		if err != nil {
			continue
		}

		for cursor.Next(ctx) {
			var idx bson.M
			if err := cursor.Decode(&idx); err != nil {
				continue
			}
			if name, ok := idx["name"].(string); ok && name == indexName {
				cursor.Close(ctx)
				return collName, nil
			}
		}
		cursor.Close(ctx)
	}

	return "", nil
}

func (m *MongoDBAdapter) IndexExists(ctx context.Context, indexName string) (bool, error) {
	collName, err := m.findCollectionForIndex(ctx, indexName)
	if err != nil {
		return false, err
	}
	return collName != "", nil
}

func (m *MongoDBAdapter) GetCurrentConfig(ctx context.Context, parameters []string) (map[string]string, error) {
	config := make(map[string]string)

	for _, param := range parameters {
		var result bson.M
		err := m.database.RunCommand(ctx, bson.D{
			{Key: "getParameter", Value: 1},
			{Key: param, Value: 1},
		}).Decode(&result)
		if err != nil {
			continue
		}
		if val, ok := result[param]; ok {
			config[param] = fmt.Sprintf("%v", val)
		}
	}

	return config, nil
}

func (m *MongoDBAdapter) SetConfig(ctx context.Context, changes map[string]string) error {
	for param, value := range changes {
		cmd := bson.D{
			{Key: "setParameter", Value: 1},
			{Key: param, Value: value},
		}
		err := m.database.RunCommand(ctx, cmd).Err()
		if err != nil {
			return fmt.Errorf("failed to set %s to %s: %w", param, value, err)
		}
	}

	return nil
}

func (m *MongoDBAdapter) GetSlowQueries(ctx context.Context, thresholdMs float64, limit int) ([]SlowQuery, error) {
	// Query from system.profile
	profileColl := m.database.Collection("system.profile")

	filter := bson.M{
		"millis": bson.M{"$gt": thresholdMs},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "millis", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := profileColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query slow queries: %w", err)
	}
	defer cursor.Close(ctx)

	var slowQueries []SlowQuery

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		queryPattern := ""
		if command, ok := doc["command"].(bson.M); ok {
			queryPattern = fmt.Sprintf("%v", command)
			if len(queryPattern) > 200 {
				queryPattern = queryPattern[:200] + "..."
			}
		}

		millis := 0.0
		if ms, ok := doc["millis"].(int32); ok {
			millis = float64(ms)
		} else if ms, ok := doc["millis"].(int64); ok {
			millis = float64(ms)
		}

		slowQueries = append(slowQueries, SlowQuery{
			QueryPattern:    queryPattern,
			ExecutionTimeMs: millis,
			CallCount:       1,
			IssueType:       "slow_query",
			Recommendation:  "Review query and add appropriate indexes",
		})
	}

	return slowQueries, nil
}

func (m *MongoDBAdapter) VacuumTable(ctx context.Context, tableName string) error {
	// MongoDB uses compact command instead of vacuum
	err := m.database.RunCommand(ctx, bson.D{
		{Key: "compact", Value: tableName},
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to compact collection %s: %w", tableName, err)
	}

	return nil
}

func (m *MongoDBAdapter) GetDeadTuples(ctx context.Context, tableName string) (int64, error) {
	// MongoDB doesn't have dead tuples concept
	// Return storage stats fragmentation as a proxy
	var stats bson.M
	err := m.database.RunCommand(ctx, bson.D{
		{Key: "collStats", Value: tableName},
	}).Decode(&stats)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection stats: %w", err)
	}

	// Return wiredTiger freeBytes as proxy for fragmentation
	if wt, ok := stats["wiredTiger"].(bson.M); ok {
		if blockManager, ok := wt["block-manager"].(bson.M); ok {
			if freeBytes, ok := blockManager["file bytes available for reuse"].(int64); ok {
				return freeBytes, nil
			}
		}
	}

	return 0, nil
}

func (m *MongoDBAdapter) TerminateQuery(ctx context.Context, pid int32, graceful bool) error {
	// MongoDB uses killOp
	err := m.database.RunCommand(ctx, bson.D{
		{Key: "killOp", Value: 1},
		{Key: "op", Value: pid},
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to kill operation: %w", err)
	}

	return nil
}

func (m *MongoDBAdapter) GetCapabilities() Capabilities {
	return Capabilities{
		SupportsIndexes:              true,
		SupportsConcurrentIndexes:    false, // MongoDB builds in background but not "concurrent" like Postgres
		SupportsUniqueIndex:          true,
		SupportsMultiColumnIndex:     true, // Compound indexes
		SupportsConfigTuning:         true,
		SupportsRuntimeConfigChanges: true,
		SupportsVacuum:               true, // Via compact command
		SupportsQueryTermination:     true,
	}
}

func (m *MongoDBAdapter) Close() error {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.client.Disconnect(ctx)
		m.client = nil
		return err
	}
	return nil
}
