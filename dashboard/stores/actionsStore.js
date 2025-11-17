class ActionsStore {
    constructor() {
        this.actions = [];
        this.maxSize = 50;
    }

    addOrUpdate(action) {
        const existingIndex = this.actions.findIndex(a => a.action_id === action.action_id);

        if (existingIndex >= 0) {
            this.actions[existingIndex] = action;
        } else {
            this.actions.unshift(action);
            if (this.actions.length > this.maxSize) {
                this.actions.pop();
            }
        }
    }

    getAll() {
        return this.actions;
    }
}

module.exports = new ActionsStore();