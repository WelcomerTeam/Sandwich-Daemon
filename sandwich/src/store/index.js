import { createStore, createLogger } from "vuex"

import dashboard from "./modules/dashboard";
import user from "./modules/user";

const debug = process.env.NODE_ENV !== 'production'

export default createStore({
    modules: {
        dashboard,
        user
    },
    strict: debug,
    plugins : debug ? [createLogger()] : []
})