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
	    }
	}

}

