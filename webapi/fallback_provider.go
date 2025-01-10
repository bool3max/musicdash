package webapi

import (
	"bool3max/musicdash/db"
	"bool3max/musicdash/music"
	"bool3max/musicdash/spotify"
	"context"
	"log"
)

// A music.ResourceProvider that prefers data from a local database, falling back to Spotify if the requested
// resource isn't preserved locally. If the "preserveIfNotFound" field is true, resources not in the dabatase
// that were obtained from Spotify will be subsequently preserved in the database.
type FallbackProvider struct {
	db      *db.Db
	spotify *spotify.Client

	preserveIfNotFound bool
}

func NewFallbackProvider(dbInstance *db.Db, spotifyClient *spotify.Client, preserveIfNotFound bool) *FallbackProvider {
	return &FallbackProvider{
		db:                 dbInstance,
		spotify:            spotifyClient,
		preserveIfNotFound: preserveIfNotFound,
	}
}

func (fp *FallbackProvider) GetTrackById(id string) (*music.Track, error) {
	fromDb, err := fp.db.GetTrackById(id)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		track, err := fp.spotify.GetTrackById(id)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			if err := track.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
				log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
			}
		}

		return track, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetSeveralTracksById(ids []string) ([]music.Track, error) {
	fromDb, err := fp.db.GetSeveralTracksById(ids)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		tracks, err := fp.spotify.GetSeveralTracksById(ids)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			// preserve all tracks
			for _, track := range tracks {
				if err := track.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
					log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
				}
			}
		}
		return tracks, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetTrackByMatch(match string) (*music.Track, error) {
	fromDb, err := fp.db.GetTrackByMatch(match)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		track, err := fp.spotify.GetTrackByMatch(match)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			if err := track.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
				log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
			}
		}

		return track, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetAlbumById(id string) (*music.Album, error) {
	fromDb, err := fp.db.GetAlbumById(id)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		album, err := fp.spotify.GetAlbumById(id)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			if err := album.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
				log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
			}
		}

		return album, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetSeveralAlbumsById(ids []string) ([]music.Album, error) {
	fromDb, err := fp.db.GetSeveralAlbumsById(ids)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		albums, err := fp.spotify.GetSeveralAlbumsById(ids)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			for _, album := range albums {
				if err := album.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
					log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
				}
			}
		}

		return albums, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetAlbumByMatch(match string) (*music.Album, error) {
	fromDb, err := fp.db.GetAlbumByMatch(match)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		log.Println("not found in DB, forwarding request to spotify")
		album, err := fp.spotify.GetAlbumByMatch(match)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			if err := album.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
				log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
			}
		}

		return album, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetArtistById(id string, discogFillLevel int, albumTypes []music.AlbumType) (*music.Artist, error) {
	fromDb, err := fp.db.GetArtistById(id, discogFillLevel, albumTypes)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		artist, err := fp.spotify.GetArtistById(id, discogFillLevel, albumTypes)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			if err := artist.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
				log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
			}
		}

		return artist, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetArtistByMatch(match string, discogFillLevel int, albumTypes []music.AlbumType) (*music.Artist, error) {
	fromDb, err := fp.db.GetArtistByMatch(match, discogFillLevel, albumTypes)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		artist, err := fp.spotify.GetArtistByMatch(match, discogFillLevel, albumTypes)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			if err := artist.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
				log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
			}
		}

		return artist, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetArtistDiscography(artist *music.Artist, albumTypes []music.AlbumType) ([]music.Album, error) {
	fromDb, err := fp.db.GetArtistDiscography(artist, albumTypes)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		discog, err := fp.spotify.GetArtistDiscography(artist, albumTypes)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			for _, album := range discog {
				if err := album.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
					log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
				}
			}
		}

		return discog, nil
	}

	// other error while trying to get resource from db
	return nil, err
}

func (fp *FallbackProvider) GetAlbumTracklist(album *music.Album) ([]music.Track, error) {
	fromDb, err := fp.db.GetAlbumTracklist(album)

	// no error, resource in db
	if err == nil {
		return fromDb, nil
	}

	// resource not in db
	if err == db.ErrResourceNotPreserved {
		tracklist, err := fp.spotify.GetAlbumTracklist(album)
		if err != nil {
			return nil, err
		}

		if fp.preserveIfNotFound {
			for _, track := range tracklist {
				if err := track.Preserve(context.Background(), fp.db.Pool(), true); err != nil {
					log.Printf("FallbackProvider: preserving resource failed: %v\n", err)
				}
			}
		}

		return tracklist, nil
	}

	// other error while trying to get resource from db
	return nil, err
}
