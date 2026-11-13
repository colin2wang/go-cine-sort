export namespace config {
	
	export class AppConfig {
	    movieFolder: string;
	    directoryFormat: string;
	    recursive: boolean;
	    archiveMove: boolean;
	    conflictStrategy: string;
	    createRatingFile: boolean;
	    createSidFile: boolean;
	    downloadPosters: boolean;
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
	        this.directoryFormat = source["directoryFormat"];
	        this.recursive = source["recursive"];
	        this.archiveMove = source["archiveMove"];
	        this.conflictStrategy = source["conflictStrategy"];
	        this.createRatingFile = source["createRatingFile"];
	        this.createSidFile = source["createSidFile"];
	        this.downloadPosters = source["downloadPosters"];
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

