class MetricsStore {
    constructor() {
        this.metrics = [];
        this.maxSize = 100;
    }

    add(metric) {
        this.metrics.unshift(metric);
        if (this.metrics.length > this.maxSize) {
            this.metrics.pop();
        }
    }

    getLatest(databaseId = null) {
        if (!databaseId) {
            return this.metrics[0] || null;
        }
        return this.metrics.find(m => 
            (m.database_id || m.DatabaseID) === databaseId
        ) || null;
    }

    getAll(databaseId = null) {
        if (!databaseId) {
            return this.metrics;
        }
        return this.metrics.filter(m => 
            (m.database_id || m.DatabaseID) === databaseId
        );
    }

    getHistory(limit = 20, databaseId = null) {
        const filtered = databaseId 
            ? this.metrics.filter(m => (m.database_id || m.DatabaseID) === databaseId)
            : this.metrics;
        return filtered.slice(0, limit);
    }

    clear() {
        this.metrics = [];
    }
}

module.exports = new MetricsStore();