class MetricsStore {
    constructor() {
        this.metrics = [];
        this.maxSize =  100;
    }

    add(metric) {
        this.metrics.unshift(metric);
        if (this.metrics.length > this.maxSize) {
            this.metrics.pop();
        }
    }

    getLatest() {
        return this.metrics[0] || null;
    }

    getAll() {
        return this.metrics;
    }

    getHistory(limit = 20) {
        return this.metrics.slice(0, limit);
    }
}

modules.export = new MetricsStore();