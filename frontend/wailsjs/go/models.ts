export namespace gitcli {
	
	export class DiscardSnapshot {
	    path: string;
	    blob: string;
	    mode: number;
	    existed: boolean;
	    undoable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscardSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.blob = source["blob"];
	        this.mode = source["mode"];
	        this.existed = source["existed"];
	        this.undoable = source["undoable"];
	    }
	}
	export class ResetUndo {
	    head: string;
	    target: string;
	    stash: string;
	
	    static createFrom(source: any = {}) {
	        return new ResetUndo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.head = source["head"];
	        this.target = source["target"];
	        this.stash = source["stash"];
	    }
	}
	export class StashEntry {
	    index: number;
	    ref: string;
	    message: string;
	    branch: string;
	    hash: string;
	    date: string;
	
	    static createFrom(source: any = {}) {
	        return new StashEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.ref = source["ref"];
	        this.message = source["message"];
	        this.branch = source["branch"];
	        this.hash = source["hash"];
	        this.date = source["date"];
	    }
	}

}

export namespace graph {
	
	export class GraphEdge {
	    fromCol: number;
	    toCol: number;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fromCol = source["fromCol"];
	        this.toCol = source["toCol"];
	        this.color = source["color"];
	    }
	}
	export class Label {
	    name: string;
	    type: string;
	    remote?: string;
	
	    static createFrom(source: any = {}) {
	        return new Label(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.remote = source["remote"];
	    }
	}
	export class GraphRow {
	    hash: string;
	    shortHash: string;
	    subject: string;
	    author: string;
	    date: string;
	    timestamp: number;
	    parentHashes: string[];
	    labels: Label[];
	    column: number;
	    color: string;
	    edgesIn: GraphEdge[];
	    edges: GraphEdge[];
	    maxColumn: number;
	
	    static createFrom(source: any = {}) {
	        return new GraphRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.shortHash = source["shortHash"];
	        this.subject = source["subject"];
	        this.author = source["author"];
	        this.date = source["date"];
	        this.timestamp = source["timestamp"];
	        this.parentHashes = source["parentHashes"];
	        this.labels = this.convertValues(source["labels"], Label);
	        this.column = source["column"];
	        this.color = source["color"];
	        this.edgesIn = this.convertValues(source["edgesIn"], GraphEdge);
	        this.edges = this.convertValues(source["edges"], GraphEdge);
	        this.maxColumn = source["maxColumn"];
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

export namespace main {
	
	export class RecentRepo {
	    path: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new RecentRepo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	    }
	}

}

export namespace repo {
	
	export class AheadBehind {
	    ahead: number;
	    behind: number;
	
	    static createFrom(source: any = {}) {
	        return new AheadBehind(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	    }
	}
	export class AppPrefs {
	    editor: string;
	    autoFetchEnabled: boolean;
	    autoFetchIntervalSecs: number;
	    commitGraphLimit: number;
	    diffContextLines: number;
	    dateFormat: string;
	    pullMode: string;
	
	    static createFrom(source: any = {}) {
	        return new AppPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.editor = source["editor"];
	        this.autoFetchEnabled = source["autoFetchEnabled"];
	        this.autoFetchIntervalSecs = source["autoFetchIntervalSecs"];
	        this.commitGraphLimit = source["commitGraphLimit"];
	        this.diffContextLines = source["diffContextLines"];
	        this.dateFormat = source["dateFormat"];
	        this.pullMode = source["pullMode"];
	    }
	}
	export class BranchInfo {
	    name: string;
	    isRemote: boolean;
	    remote: string;
	    isCurrent: boolean;
	    hash: string;
	
	    static createFrom(source: any = {}) {
	        return new BranchInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isRemote = source["isRemote"];
	        this.remote = source["remote"];
	        this.isCurrent = source["isCurrent"];
	        this.hash = source["hash"];
	    }
	}
	export class ConflictContent {
	    base: string;
	    ours: string;
	    theirs: string;
	    working: string;
	
	    static createFrom(source: any = {}) {
	        return new ConflictContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base = source["base"];
	        this.ours = source["ours"];
	        this.theirs = source["theirs"];
	        this.working = source["working"];
	    }
	}
	export class HunkLine {
	    type: string;
	    content: string;
	    oldLine: number;
	    newLine: number;
	
	    static createFrom(source: any = {}) {
	        return new HunkLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.content = source["content"];
	        this.oldLine = source["oldLine"];
	        this.newLine = source["newLine"];
	    }
	}
	export class Hunk {
	    header: string;
	    lines: HunkLine[];
	
	    static createFrom(source: any = {}) {
	        return new Hunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.header = source["header"];
	        this.lines = this.convertValues(source["lines"], HunkLine);
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
	export class FileDiff {
	    oldPath: string;
	    newPath: string;
	    hunks: Hunk[];
	    isBinary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oldPath = source["oldPath"];
	        this.newPath = source["newPath"];
	        this.hunks = this.convertValues(source["hunks"], Hunk);
	        this.isBinary = source["isBinary"];
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
	export class FileStatus {
	    path: string;
	    oldPath: string;
	    staged: string;
	    unstaged: string;
	
	    static createFrom(source: any = {}) {
	        return new FileStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.oldPath = source["oldPath"];
	        this.staged = source["staged"];
	        this.unstaged = source["unstaged"];
	    }
	}
	
	
	export class RebaseState {
	    step: number;
	    total: number;
	    message: string;
	    onto: string;
	
	    static createFrom(source: any = {}) {
	        return new RebaseState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.total = source["total"];
	        this.message = source["message"];
	        this.onto = source["onto"];
	    }
	}
	export class SessionTab {
	    path: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionTab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	    }
	}
	export class Session {
	    tabs: SessionTab[];
	    activeIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabs = this.convertValues(source["tabs"], SessionTab);
	        this.activeIndex = source["activeIndex"];
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
	
	export class SwitchResult {
	    success: boolean;
	    stashUsed: boolean;
	    conflictFiles: string[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SwitchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.stashUsed = source["stashUsed"];
	        this.conflictFiles = source["conflictFiles"];
	        this.message = source["message"];
	    }
	}

}

