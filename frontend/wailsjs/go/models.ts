export namespace main {
	
	export class APIStatus {
	    running: boolean;
	    port: number;
	    url: string;
	    tls: boolean;
	    fingerprint: string;
	
	    static createFrom(source: any = {}) {
	        return new APIStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.url = source["url"];
	        this.tls = source["tls"];
	        this.fingerprint = source["fingerprint"];
	    }
	}
	export class UpdateInfo {
	    checking: boolean;
	    installing: boolean;
	    progress: string;
	    available: boolean;
	    version: string;
	    notes: string;
	    current: string;
	    checkedAt: string;
	    error: string;
	    notify: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checking = source["checking"];
	        this.installing = source["installing"];
	        this.progress = source["progress"];
	        this.available = source["available"];
	        this.version = source["version"];
	        this.notes = source["notes"];
	        this.current = source["current"];
	        this.checkedAt = source["checkedAt"];
	        this.error = source["error"];
	        this.notify = source["notify"];
	    }
	}

}

export namespace settings {
	
	export class Settings {
	    theme: string;
	    opacity: number;
	    apiAutoStart: boolean;
	    apiPort: number;
	    apiKey: string;
	    apiAllowlist: string[];
	    apiHttps: boolean;
	    updateAutoCheck: boolean;
	    updateSkippedVersion: string;
	    updateLaterUntil: string;
	    updateLastAutoCheck: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.opacity = source["opacity"];
	        this.apiAutoStart = source["apiAutoStart"];
	        this.apiPort = source["apiPort"];
	        this.apiKey = source["apiKey"];
	        this.apiAllowlist = source["apiAllowlist"];
	        this.apiHttps = source["apiHttps"];
	        this.updateAutoCheck = source["updateAutoCheck"];
	        this.updateSkippedVersion = source["updateSkippedVersion"];
	        this.updateLaterUntil = source["updateLaterUntil"];
	        this.updateLastAutoCheck = source["updateLastAutoCheck"];
	    }
	}

}

