export namespace main {
	
	export class ApplyResult {
	    ok: boolean;
	    error?: string;
	    recordCount: number;
	    highStressN: number;
	    incompleteN: number;
	
	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.recordCount = source["recordCount"];
	        this.highStressN = source["highStressN"];
	        this.incompleteN = source["incompleteN"];
	    }
	}
	export class AutoDetectResult {
	    nbjsq_questions: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new AutoDetectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nbjsq_questions = source["nbjsq_questions"];
	    }
	}
	export class BasicAttributesMap {
	    empid: string;
	    age: string;
	    gender: string;
	    dept1: string;
	    dept2: string;
	
	    static createFrom(source: any = {}) {
	        return new BasicAttributesMap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.empid = source["empid"];
	        this.age = source["age"];
	        this.gender = source["gender"];
	        this.dept1 = source["dept1"];
	        this.dept2 = source["dept2"];
	    }
	}
	export class CSVLoadResult {
	    ok: boolean;
	    error?: string;
	    fileName: string;
	    rowCount: number;
	    colCount: number;
	    headers: string[];
	    preview: string[][];
	    uniqueVals: string[];
	
	    static createFrom(source: any = {}) {
	        return new CSVLoadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.fileName = source["fileName"];
	        this.rowCount = source["rowCount"];
	        this.colCount = source["colCount"];
	        this.headers = source["headers"];
	        this.preview = source["preview"];
	        this.uniqueVals = source["uniqueVals"];
	    }
	}
	export class ColumnMapConfig {
	    basic_attributes: BasicAttributesMap;
	    nbjsq_questions: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ColumnMapConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.basic_attributes = this.convertValues(source["basic_attributes"], BasicAttributesMap);
	        this.nbjsq_questions = source["nbjsq_questions"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GenderValMap {
	    male: string;
	    female: string;
	
	    static createFrom(source: any = {}) {
	        return new GenderValMap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.male = source["male"];
	        this.female = source["female"];
	    }
	}
	export class NBJSQBulkValMap {
	    group_aefgh: string[];
	    group_b: string[];
	    group_c: string[];
	    group_d: string[];
	
	    static createFrom(source: any = {}) {
	        return new NBJSQBulkValMap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.group_aefgh = source["group_aefgh"];
	        this.group_b = source["group_b"];
	        this.group_c = source["group_c"];
	        this.group_d = source["group_d"];
	    }
	}
	export class ValueMapConfig {
	    gender: GenderValMap;
	    nbjsq_bulk: NBJSQBulkValMap;
	    nbjsq_individual: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new ValueMapConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gender = this.convertValues(source["gender"], GenderValMap);
	        this.nbjsq_bulk = this.convertValues(source["nbjsq_bulk"], NBJSQBulkValMap);
	        this.nbjsq_individual = source["nbjsq_individual"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

