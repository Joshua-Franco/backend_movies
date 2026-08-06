package models

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Movie struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	PosterPath    *string `json:"poster_path"`
	BackdropPath  *string `json:"backdrop_path"`
	ReleaseDate   string  `json:"release_date"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	GenreIDs      []int   `json:"genre_ids"`
	Popularity    float64 `json:"popularity"`
	Adult         bool    `json:"adult"`
	MediaType     string  `json:"media_type,omitempty"`
}

type TVShow struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	OriginalName  string  `json:"original_name"`
	Overview      string  `json:"overview"`
	PosterPath    *string `json:"poster_path"`
	BackdropPath  *string `json:"backdrop_path"`
	FirstAirDate  string  `json:"first_air_date"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	GenreIDs      []int   `json:"genre_ids"`
	Popularity    float64 `json:"popularity"`
	MediaType     string  `json:"media_type,omitempty"`
}

type Video struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Site string `json:"site"`
	Type string `json:"type"`
}

type CastMember struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Character   string  `json:"character"`
	ProfilePath *string `json:"profile_path"`
	Order       int     `json:"order"`
}

type CrewMember struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Job         string  `json:"job"`
	Department  string  `json:"department"`
	ProfilePath *string `json:"profile_path"`
}

type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type VideoResults struct {
	Results []Video `json:"results"`
}

type MovieDetail struct {
	ID            int          `json:"id"`
	Title         string       `json:"title"`
	OriginalTitle string       `json:"original_title"`
	Overview      string       `json:"overview"`
	PosterPath    *string      `json:"poster_path"`
	BackdropPath  *string      `json:"backdrop_path"`
	ReleaseDate   string       `json:"release_date"`
	Runtime       int          `json:"runtime"`
	VoteAverage   float64      `json:"vote_average"`
	VoteCount     int          `json:"vote_count"`
	Genres        []Genre      `json:"genres"`
	Tagline       string       `json:"tagline"`
	Status        string       `json:"status"`
	Budget        int64        `json:"budget"`
	Revenue       int64        `json:"revenue"`
	Homepage      *string      `json:"homepage"`
	Videos        *VideoResults `json:"videos,omitempty"`
	Credits       *Credits     `json:"credits,omitempty"`
}

type TMDBResponse struct {
	Page            int         `json:"page"`
	Results         interface{} `json:"results"`
	TotalPages      int         `json:"total_pages"`
	TotalResults    int         `json:"total_results"`
}

type GenreList struct {
	Genres []Genre `json:"genres"`
}
