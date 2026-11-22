export namespace cache {
	
	export class Entry {
	    key: string;
	    method: string;
	    url: string;
	    source: string;
	    purpose: string;
	    statusCode: number;
	    header: Record<string, string>;
	    body: string;
	    bodyBytes: number;
	    // Go type: time
	    cachedAt: any;
	    // Go type: time
	    expiresAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.source = source["source"];
	        this.purpose = source["purpose"];
	        this.statusCode = source["statusCode"];
	        this.header = source["header"];
	        this.body = source["body"];
	        this.bodyBytes = source["bodyBytes"];
	        this.cachedAt = this.convertValues(source["cachedAt"], null);
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
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
	export class Stats {
	    count: number;
	    totalBodyBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.totalBodyBytes = source["totalBodyBytes"];
	    }
	}
	export class Summary {
	    key: string;
	    method: string;
	    url: string;
	    source: string;
	    purpose: string;
	    statusCode: number;
	    bodyBytes: number;
	    // Go type: time
	    cachedAt: any;
	    // Go type: time
	    expiresAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.source = source["source"];
	        this.purpose = source["purpose"];
	        this.statusCode = source["statusCode"];
	        this.bodyBytes = source["bodyBytes"];
	        this.cachedAt = this.convertValues(source["cachedAt"], null);
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
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

export namespace config {
	
	export class AppConfig {
	    movieFolder: string;
	    youtubeSourceDir: string;
	    youtubeTargetDir: string;
	    directoryFormat: string;
	    recursive: boolean;
	    preferSourceDir: boolean;
	    archiveMove: boolean;
	    conflictStrategy: string;
	    createRatingFile: boolean;
	    createSidFile: boolean;
	    downloadPosters: boolean;
	    posterFormat: string;
	    fetchImdb: boolean;
	    extensions: string[];
	    cleanupPatterns: string[];
	    techPatterns: string[];
	    languagePatterns: string[];
	    configPath: string;
	    dataDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.movieFolder = source["movieFolder"];
	        this.youtubeSourceDir = source["youtubeSourceDir"];
	        this.youtubeTargetDir = source["youtubeTargetDir"];
	        this.directoryFormat = source["directoryFormat"];
	        this.recursive = source["recursive"];
	        this.preferSourceDir = source["preferSourceDir"];
	        this.archiveMove = source["archiveMove"];
	        this.conflictStrategy = source["conflictStrategy"];
	        this.createRatingFile = source["createRatingFile"];
	        this.createSidFile = source["createSidFile"];
	        this.downloadPosters = source["downloadPosters"];
	        this.posterFormat = source["posterFormat"];
	        this.fetchImdb = source["fetchImdb"];
	        this.extensions = source["extensions"];
	        this.cleanupPatterns = source["cleanupPatterns"];
	        this.techPatterns = source["techPatterns"];
	        this.languagePatterns = source["languagePatterns"];
	        this.configPath = source["configPath"];
	        this.dataDir = source["dataDir"];
	    }
	}

}

export namespace model {
	
	export class HistoryStats {
	    total: number;
	    averageRating: number;
	    posterTotal: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.averageRating = source["averageRating"];
	        this.posterTotal = source["posterTotal"];
	    }
	}
	export class PosterRecord {
	    id: number;
	    posterId: string;
	    title: string;
	    keyword: string;
	    region: string;
	    thumbnailUrl: string;
	    originalUrl: string;
	    fileSize: number;
	    isDownloaded: boolean;
	    localPath: string;
	
	    static createFrom(source: any = {}) {
	        return new PosterRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.posterId = source["posterId"];
	        this.title = source["title"];
	        this.keyword = source["keyword"];
	        this.region = source["region"];
	        this.thumbnailUrl = source["thumbnailUrl"];
	        this.originalUrl = source["originalUrl"];
	        this.fileSize = source["fileSize"];
	        this.isDownloaded = source["isDownloaded"];
	        this.localPath = source["localPath"];
	    }
	}
	export class MovieDetail {
	    id: number;
	    sid: string;
	    title: string;
	    originalTitle: string;
	    year: string;
	    rating: string;
	    ratingCount: number;
	    imdbId: string;
	    directoryPath: string;
	    posterUrl: string;
	    createdAt: string;
	    posterCount: number;
	    aliases: string;
	    description: string;
	    country: string;
	    language: string;
	    runtime: number;
	    top250Rank: number;
	    shortCommentsCount: number;
	    reviewsCount: number;
	    directors: string[];
	    actors: string[];
	    genres: string[];
	    releaseDates: string[];
	    awards: string[];
	    similarMovies: string[];
	    posters: PosterRecord[];
	
	    static createFrom(source: any = {}) {
	        return new MovieDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sid = source["sid"];
	        this.title = source["title"];
	        this.originalTitle = source["originalTitle"];
	        this.year = source["year"];
	        this.rating = source["rating"];
	        this.ratingCount = source["ratingCount"];
	        this.imdbId = source["imdbId"];
	        this.directoryPath = source["directoryPath"];
	        this.posterUrl = source["posterUrl"];
	        this.createdAt = source["createdAt"];
	        this.posterCount = source["posterCount"];
	        this.aliases = source["aliases"];
	        this.description = source["description"];
	        this.country = source["country"];
	        this.language = source["language"];
	        this.runtime = source["runtime"];
	        this.top250Rank = source["top250Rank"];
	        this.shortCommentsCount = source["shortCommentsCount"];
	        this.reviewsCount = source["reviewsCount"];
	        this.directors = source["directors"];
	        this.actors = source["actors"];
	        this.genres = source["genres"];
	        this.releaseDates = source["releaseDates"];
	        this.awards = source["awards"];
	        this.similarMovies = source["similarMovies"];
	        this.posters = this.convertValues(source["posters"], PosterRecord);
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
	export class MovieFileInfo {
	    filePath: string;
	    movieName: string;
	    year: string;
	    extension: string;
	    rawFilename: string;
	
	    static createFrom(source: any = {}) {
	        return new MovieFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.movieName = source["movieName"];
	        this.year = source["year"];
	        this.extension = source["extension"];
	        this.rawFilename = source["rawFilename"];
	    }
	}
	export class MovieRecord {
	    id: number;
	    sid: string;
	    title: string;
	    originalTitle: string;
	    year: string;
	    rating: string;
	    ratingCount: number;
	    imdbId: string;
	    directoryPath: string;
	    posterUrl: string;
	    createdAt: string;
	    posterCount: number;
	
	    static createFrom(source: any = {}) {
	        return new MovieRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sid = source["sid"];
	        this.title = source["title"];
	        this.originalTitle = source["originalTitle"];
	        this.year = source["year"];
	        this.rating = source["rating"];
	        this.ratingCount = source["ratingCount"];
	        this.imdbId = source["imdbId"];
	        this.directoryPath = source["directoryPath"];
	        this.posterUrl = source["posterUrl"];
	        this.createdAt = source["createdAt"];
	        this.posterCount = source["posterCount"];
	    }
	}
	
	export class ProcessResult {
	    filename: string;
	    filePath: string;
	    movieName: string;
	    year: string;
	    directory: string;
	    videoPath: string;
	    error: string;
	    success: boolean;
	    skipped: boolean;
	    sid: string;
	    title: string;
	    rating: string;
	    posterCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.filePath = source["filePath"];
	        this.movieName = source["movieName"];
	        this.year = source["year"];
	        this.directory = source["directory"];
	        this.videoPath = source["videoPath"];
	        this.error = source["error"];
	        this.success = source["success"];
	        this.skipped = source["skipped"];
	        this.sid = source["sid"];
	        this.title = source["title"];
	        this.rating = source["rating"];
	        this.posterCount = source["posterCount"];
	    }
	}

}

export namespace service {
	
	export class MediaFileMeta {
	    path: string;
	    dir: string;
	    filename: string;
	    sizeMB: number;
	    title: string;
	    comment: string;
	    newName: string;
	    renamable: boolean;
	    taggable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaFileMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.dir = source["dir"];
	        this.filename = source["filename"];
	        this.sizeMB = source["sizeMB"];
	        this.title = source["title"];
	        this.comment = source["comment"];
	        this.newName = source["newName"];
	        this.renamable = source["renamable"];
	        this.taggable = source["taggable"];
	    }
	}

}

export namespace youtube {
	
	export class MovePlan {
	    filename: string;
	    sourceDir: string;
	    uploader: string;
	    destDir: string;
	    destPath: string;
	
	    static createFrom(source: any = {}) {
	        return new MovePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.sourceDir = source["sourceDir"];
	        this.uploader = source["uploader"];
	        this.destDir = source["destDir"];
	        this.destPath = source["destPath"];
	    }
	}
	export class PlanResult {
	    moves: MovePlan[];
	    conflicts: string[];
	    unmatched: string[];
	
	    static createFrom(source: any = {}) {
	        return new PlanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.moves = this.convertValues(source["moves"], MovePlan);
	        this.conflicts = source["conflicts"];
	        this.unmatched = source["unmatched"];
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

