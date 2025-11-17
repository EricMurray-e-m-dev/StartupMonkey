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

    getAll() {
        return this.detections;
    }
}

module.exports = new DetectionsStore();