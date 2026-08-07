package routes

import (
	"backend_movies/handlers"
	"backend_movies/middleware"
	"backend_movies/services"

	"github.com/go-chi/chi/v5"
)

func SetupRoutes(tmdb *services.TMDBClient, cache *services.CacheService, r2 *services.R2Client) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(middleware.Logger)

	r.Get("/healthz", handlers.HealthCheck)

	trendingHandler := handlers.NewTrendingHandler(tmdb, cache)
	popularHandler := handlers.NewPopularHandler(tmdb, cache)
	detailHandler := handlers.NewDetailHandler(tmdb, cache)
	searchHandler := handlers.NewSearchHandler(tmdb, cache)
	genresHandler := handlers.NewGenresHandler(tmdb, cache)
	userHandler := handlers.NewUserHandler()
	discoverHandler := handlers.NewDiscoverHandler(tmdb, cache)
	personHandler := handlers.NewPersonHandler(tmdb, cache)
	streamHandler := handlers.NewStreamHandler(r2)

	r.Route("/api", func(r chi.Router) {
		r.Get("/trending/{mediaType}", trendingHandler.GetTrending)

		r.Get("/popular/{mediaType}", popularHandler.GetPopular)
		r.Get("/top-rated/{mediaType}", popularHandler.GetTopRated)
		r.Get("/now-playing/{mediaType}", popularHandler.GetNowPlaying)

		r.Get("/search", searchHandler.Search)

		r.Get("/genres/{mediaType}", genresHandler.GetGenres)
		r.Get("/discover/movie", genresHandler.DiscoverByGenre)
		r.Get("/discover/companies", discoverHandler.DiscoverByCompanies)
		r.Get("/discover/tv-companies", discoverHandler.DiscoverTVByCompanies)

		r.Get("/person/{personId}", personHandler.GetPerson)

		r.Get("/stream/{tmdbId}", streamHandler.GetStreamInfo)

		r.Route("/my-list", func(r chi.Router) {
			r.Get("/", userHandler.GetMyList)
			r.Post("/", userHandler.AddToList)
			r.Delete("/{tmdbId}", userHandler.RemoveFromList)
		})

		r.Route("/continue-watching", func(r chi.Router) {
			r.Get("/", userHandler.GetContinueWatching)
			r.Post("/", userHandler.UpdateProgress)
			r.Delete("/{tmdbId}", userHandler.RemoveFromContinueWatching)
		})

		r.Get("/{mediaType}/{id}", detailHandler.GetDetail)
	})

	r.Get("/hls/{movieName}/master.m3u8", streamHandler.ServeMasterPlaylist)
	r.Get("/hls/{movieName}/{quality}/playlist.m3u8", streamHandler.ServeQualityPlaylist)
	r.Get("/hls/{movieName}/{quality}/{segment}.ts", streamHandler.ServeSegment)

	return r
}
