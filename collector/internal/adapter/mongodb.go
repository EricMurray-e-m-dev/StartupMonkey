package adapter

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBAdapter struct {
	connectionString string
	databaseID       string
	databaseName     string
	client           *mongo.Client
	database         *mongo.Database
}

func NewMongoDBAdapter(connectionString, databaseID string) *MongoDBAdapter {
	dbName := extractMongoDBName(connectionString)

	return &MongoDBAdapter{
		connectionString: connectionString,
		databaseID:       databaseID,
		databaseName:     dbName,
	}
}

func extractMongoDBName(connStr string) string {
	// mongodb://user:pass@host:port/dbname?options
	if idx := strings.Index(connStr, "?"); idx != -1 {
		connStr = connStr[:idx]
	}
	parts := strings.Split(connStr, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "test"
}

func (m *MongoDBAdapter) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(m.connectionString).
		SetMaxPoolSize(5).
		SetMinPoolSize(1).
		SetMaxConnIdleTime(5 * time.Minute)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	m.database = client.Database(m.databaseName)

	log.Printf("Connected to MongoDB: %s (database: %s)", m.databaseID, m.databaseName)
	return nil
}

func (m *MongoDBAdapter) CollectMetrics() (*RawMetrics, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	ctx := context.Background()
	metrics := NewRawMetrics(m.databaseID, "mongodb")

	// Server status for connections and cache
	if err := m.collectServerStatus(ctx, metrics); err != nil {
		log.Printf("Warning: failed to get server status: %v", err)
	}

	// Collection scan stats
	if err := m.collectCollectionScans(ctx, metrics); err != nil {
		log.Printf("Warning: failed to get collection scans: %v", err)
	}

	// Database stats
	if err := m.collectDatabaseStats(ctx, metrics); err != nil {
		log.Printf("Warning: failed to get database stats: %v", err)
	}

	return metrics, nil
}

func (m *MongoDBAdapter) collectServerStatus(ctx context.Context, metrics *RawMetrics) error {
	var result bson.M
	err := m.database.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
	if err != nil {
		return err
	}

	// Connections
	if connections, ok := result["connections"].(bson.M); ok {
		if current, ok := connections["current"].(int32); ok {
			active := int32(current)
			metrics.Connections = &ConnectionMetrics{
				Active: &active,
			}
		}
		if available, ok := connections["available"].(int32); ok {
			if metrics.Connections != nil && metrics.Connections.Active != nil {
				maxConn := *metrics.Connections.Active + int32(available)
				metrics.Connections.Max = &maxConn
			}
		}
	}

	// WiredTiger cache
	if wt, ok := result["wiredTiger"].(bson.M); ok {
		if cache, ok := wt["cache"].(bson.M); ok {
			if pagesRead, ok := cache["pages read into cache"].(int64); ok {
				if pagesRequested, ok := cache["pages requested from the cache"].(int64); ok {
					if pagesRequested > 0 {
						hitRate := float64(pagesRequested-pagesRead) / float64(pagesRequested)
						if hitRate < 0 {
							hitRate = 0
						}
						metrics.Cache = &CacheMetrics{
							HitRate: &hitRate,
						}
					}
				}
			}

			if bytesInCache, ok := cache["bytes currently in the cache"].(int64); ok {
				metrics.ExtendedMetrics["mongodb.cache_bytes_in_cache"] = float64(bytesInCache)
			}
		}
	}

	// Global lock / active clients
	if globalLock, ok := result["globalLock"].(bson.M); ok {
		if activeClients, ok := globalLock["activeClients"].(bson.M); ok {
			if readers, ok := activeClients["readers"].(int32); ok {
				metrics.ExtendedMetrics["mongodb.active_readers"] = float64(readers)
			}
			if writers, ok := activeClients["writers"].(int32); ok {
				metrics.ExtendedMetrics["mongodb.active_writers"] = float64(writers)
			}
		}
	}

	// Operation counters
	if opcounters, ok := result["opcounters"].(bson.M); ok {
		if query, ok := opcounters["query"].(int64); ok {
			metrics.ExtendedMetrics["mongodb.opcounters_query"] = float64(query)
		}
		if insert, ok := opcounters["insert"].(int64); ok {
			metrics.ExtendedMetrics["mongodb.opcounters_insert"] = float64(insert)
		}
		if update, ok := opcounters["update"].(int64); ok {
			metrics.ExtendedMetrics["mongodb.opcounters_update"] = float64(update)
		}
		if del, ok := opcounters["delete"].(int64); ok {
			metrics.ExtendedMetrics["mongodb.opcounters_delete"] = float64(del)
		}
	}

	return nil
}

func (m *MongoDBAdapter) collectCollectionScans(ctx context.Context, metrics *RawMetrics) error {
	// Try profile collection first (requires --profile 2)
	profileColl := m.database.Collection("system.profile")

	filter := bson.M{
		"op":          bson.M{"$in": []string{"query", "find"}},
		"planSummary": bson.M{"$regex": "COLLSCAN"},
	}

	collScanCount, err := profileColl.CountDocuments(ctx, filter)
	if err != nil {
		// Profiling not enabled - try collStats approach
		return m.collectCollectionScansFromCollStats(ctx, metrics)
	}

	seqScans := int32(collScanCount)
	metrics.Queries = &QueryMetrics{
		SequentialScans: &seqScans,
	}
	metrics.ExtendedMetrics["mongodb.total_collscans"] = float64(collScanCount)

	// Find worst collection
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$ns",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
		{{Key: "$limit", Value: 1}},
	}

	cursor, err := profileColl.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err == nil && result.ID != "" {
			parts := strings.Split(result.ID, ".")
			if len(parts) >= 2 {
				collName := parts[len(parts)-1]
				metrics.Labels["mongodb.worst_seq_scan_table"] = collName
				metrics.ExtendedMetrics[fmt.Sprintf("mongodb.table.%s.seq_scans", collName)] = float64(result.Count)

				m.identifyMissingIndex(ctx, collName, metrics)
			}
		}
	}

	return nil
}

func (m *MongoDBAdapter) collectCollectionScansFromCollStats(ctx context.Context, metrics *RawMetrics) error {
	collections, err := m.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return err
	}

	var worstCollection string
	var maxScans int64
	var totalScans int64

	for _, collName := range collections {
		if strings.HasPrefix(collName, "system.") {
			continue
		}

		// Get index stats
		pipeline := mongo.Pipeline{
			{{Key: "$indexStats", Value: bson.M{}}},
		}

		cursor, err := m.database.Collection(collName).Aggregate(ctx, pipeline)
		if err != nil {
			continue
		}

		var totalIndexAccesses int64
		for cursor.Next(ctx) {
			var indexStat struct {
				Accesses struct {
					Ops int64 `bson:"ops"`
				} `bson:"accesses"`
			}
			if err := cursor.Decode(&indexStat); err == nil {
				totalIndexAccesses += indexStat.Accesses.Ops
			}
		}
		cursor.Close(ctx)

		count, _ := m.database.Collection(collName).EstimatedDocumentCount(ctx)

		// Low index usage relative to doc count = likely collection scans
		if count > 1000 && totalIndexAccesses < count/10 {
			scanEstimate := count - totalIndexAccesses
			totalScans += scanEstimate

			if scanEstimate > maxScans {
				maxScans = scanEstimate
				worstCollection = collName
			}

			metrics.ExtendedMetrics[fmt.Sprintf("mongodb.table.%s.seq_scans", collName)] = float64(scanEstimate)
			metrics.ExtendedMetrics[fmt.Sprintf("mongodb.table.%s.idx_scans", collName)] = float64(totalIndexAccesses)
		}
	}

	if worstCollection != "" {
		seqScans := int32(totalScans)
		metrics.Queries = &QueryMetrics{
			SequentialScans: &seqScans,
		}
		metrics.Labels["mongodb.worst_seq_scan_table"] = worstCollection
		m.identifyMissingIndex(ctx, worstCollection, metrics)
	}

	return nil
}

func (m *MongoDBAdapter) identifyMissingIndex(ctx context.Context, collName string, metrics *RawMetrics) {
	// Check profile for query patterns
	profileColl := m.database.Collection("system.profile")

	filter := bson.M{
		"ns":          fmt.Sprintf("%s.%s", m.databaseName, collName),
		"planSummary": bson.M{"$regex": "COLLSCAN"},
	}

	cursor, err := profileColl.Find(ctx, filter, options.Find().SetLimit(10))
	if err != nil {
		m.guessIndexField(ctx, collName, metrics)
		return
	}
	defer cursor.Close(ctx)

	fieldCounts := make(map[string]int)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		if command, ok := doc["command"].(bson.M); ok {
			if filterDoc, ok := command["filter"].(bson.M); ok {
				for field := range filterDoc {
					if !strings.HasPrefix(field, "$") {
						fieldCounts[field]++
					}
				}
			}
		}
	}

	var bestField string
	var maxCount int
	for field, count := range fieldCounts {
		if count > maxCount {
			maxCount = count
			bestField = field
		}
	}

	if bestField != "" {
		metrics.Labels["mongodb.recommended_index_column"] = bestField
	} else {
		m.guessIndexField(ctx, collName, metrics)
	}
}

func (m *MongoDBAdapter) guessIndexField(ctx context.Context, collName string, metrics *RawMetrics) {
	var doc bson.M
	err := m.database.Collection(collName).FindOne(ctx, bson.M{}).Decode(&doc)
	if err != nil {
		return
	}

	// Get existing indexes
	cursor, err := m.database.Collection(collName).Indexes().List(ctx)
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	indexedFields := make(map[string]bool)
	for cursor.Next(ctx) {
		var idx bson.M
		if err := cursor.Decode(&idx); err != nil {
			continue
		}
		if key, ok := idx["key"].(bson.M); ok {
			for field := range key {
				indexedFields[field] = true
			}
		}
	}

	// Find field that looks like foreign key and isn't indexed
	for field := range doc {
		if field == "_id" {
			continue
		}
		if strings.HasSuffix(field, "_id") && !indexedFields[field] {
			metrics.Labels["mongodb.recommended_index_column"] = field
			return
		}
	}

	// Fallback: any non-indexed field
	for field := range doc {
		if field == "_id" {
			continue
		}
		if !indexedFields[field] {
			metrics.Labels["mongodb.recommended_index_column"] = field
			return
		}
	}
}

func (m *MongoDBAdapter) collectDatabaseStats(ctx context.Context, metrics *RawMetrics) error {
	var stats bson.M
	err := m.database.RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&stats)
	if err != nil {
		return err
	}

	if dataSize, ok := stats["dataSize"].(int64); ok {
		metrics.Storage = &StorageMetrics{
			UsedSizeBytes: &dataSize,
		}
		metrics.ExtendedMetrics["mongodb.database_size_bytes"] = float64(dataSize)
		metrics.ExtendedMetrics["mongodb.database_size_mb"] = float64(dataSize) / 1024 / 1024
	} else if dataSize, ok := stats["dataSize"].(float64); ok {
		size := int64(dataSize)
		metrics.Storage = &StorageMetrics{
			UsedSizeBytes: &size,
		}
		metrics.ExtendedMetrics["mongodb.database_size_bytes"] = dataSize
		metrics.ExtendedMetrics["mongodb.database_size_mb"] = dataSize / 1024 / 1024
	}

	if indexSize, ok := stats["indexSize"].(int64); ok {
		metrics.ExtendedMetrics["mongodb.index_size_bytes"] = float64(indexSize)
	}

	if collections, ok := stats["collections"].(int32); ok {
		metrics.ExtendedMetrics["mongodb.collection_count"] = float64(collections)
	}

	return nil
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

func (m *MongoDBAdapter) HealthCheck() error {
	if m.client == nil {
		return ErrNotConnected
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Ping(ctx, nil)
}

func (m *MongoDBAdapter) GetUnavailableFeatures() []string {
	return []string{"query_statistics"}
}
