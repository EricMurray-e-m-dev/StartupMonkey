class DetectionsStore {
    constructor() {
        this.detections = [];
        this.maxSize = 50;
    }

    add(detection) {
        const exists = this.detections.some(d => d.id === detection.id);
        if (!exists) {
            this.detections.unshift(detection);
            if (this.detections.length > this.maxSize) {
                this.detections.pop();
            }
        }
    }

    getAll(databaseId = null) {
        if (!databaseId) {
            return this.detections;
        }
        return this.detections.filter(d => d.database_id === databaseId);
    }

    clear() {
        this.detections = [];
    }
}

module.exports = new DetectionsStore();