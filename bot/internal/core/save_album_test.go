package core

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mellomaths/lifesoundtrack/bot/internal/store"
)

type fakeSearch struct {
	cands []AlbumCandidate
	err   error
}

func (f *fakeSearch) Search(ctx context.Context, query string) ([]AlbumCandidate, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.cands, nil
}

type memStore struct {
	listenerID        string
	sessionID         string
	disambigRaw       []byte
	insertCalls       int
	deleteSessCalls   int
	deleteListenerDis int
}

func (m *memStore) UpsertListener(ctx context.Context, source, externalID, displayName, username string) (*store.Listener, error) {
	if m.listenerID == "" {
		m.listenerID = "11111111-1111-1111-1111-111111111111"
	}
	return &store.Listener{ID: m.listenerID}, nil
}

func (m *memStore) DeleteDisambigForListener(ctx context.Context, listenerID string) error {
	m.deleteListenerDis++
	return nil
}

func (m *memStore) CreateDisambiguationSession(ctx context.Context, listenerID string, candidatesJSON []byte, ttl time.Duration) (string, error) {
	m.disambigRaw = append([]byte(nil), candidatesJSON...)
	if m.sessionID == "" {
		m.sessionID = "22222222-2222-2222-2222-222222222222"
	}
	return m.sessionID, nil
}

func (m *memStore) InsertSavedAlbum(ctx context.Context, p store.InsertSavedAlbumParams) (string, error) {
	m.insertCalls++
	return "album-id", nil
}

func (m *memStore) LatestOpenDisambiguationSession(ctx context.Context, source, externalID string) (*store.Session, []byte, error) {
	if len(m.disambigRaw) == 0 {
		return nil, nil, nil
	}
	return &store.Session{ID: m.sessionID}, m.disambigRaw, nil
}

func (m *memStore) DeleteDisambiguationSession(ctx context.Context, sessionID string) error {
	m.deleteSessCalls++
	m.disambigRaw = nil
	return nil
}

func TestProcessAlbumQuery_Empty(t *testing.T) {
	t.Parallel()
	svc := &SaveService{Store: &memStore{}, Search: &fakeSearch{}}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeEmptyQuery {
		t.Fatalf("outcome %v", um.Outcome)
	}
}

func longQuery(nRunes int) string {
	b := make([]rune, nRunes)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}

func TestProcessAlbumQuery_TooLong(t *testing.T) {
	t.Parallel()
	svc := &SaveService{Store: &memStore{}, Search: &fakeSearch{}}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", longQuery(MaxQueryRunes+1))
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeTooLong {
		t.Fatalf("outcome %v", um.Outcome)
	}
}

func TestProcessAlbumQuery_SingleMatchSaves(t *testing.T) {
	t.Parallel()
	y := 2012
	st := &memStore{}
	svc := &SaveService{
		Store: st,
		Search: &fakeSearch{cands: []AlbumCandidate{{
			Title: "Red", PrimaryArtist: "Taylor Swift", Year: &y,
			Provider: "x", ProviderRef: "r1",
		}}},
	}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "red")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeSaved {
		t.Fatalf("outcome %v text %q", um.Outcome, um.Text)
	}
	if st.insertCalls != 1 {
		t.Fatalf("inserts %d", st.insertCalls)
	}
}

func TestProcessAlbumQuery_SingleMatch_SpotifyProvider(t *testing.T) {
	t.Parallel()
	y := 1971
	st := &memStore{}
	svc := &SaveService{
		Store: st,
		Search: &fakeSearch{cands: []AlbumCandidate{{
			Title: "Abbey Road", PrimaryArtist: "The Beatles", Year: &y,
			Provider: "spotify", ProviderRef: "spotify-album-1",
		}}},
	}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "abbey road")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeSaved {
		t.Fatalf("outcome %v", um.Outcome)
	}
	if st.insertCalls != 1 {
		t.Fatalf("inserts %d", st.insertCalls)
	}
}

func TestProcessAlbumQuery_DuplicateUserVisibleLabelSavesFirst(t *testing.T) {
	t.Parallel()
	y := 2015
	st := &memStore{}
	// Two raw rows with identical "Title | Artist (Year)" but different provider refs (e.g. two catalogs).
	svc := &SaveService{
		Store: st,
		Search: &fakeSearch{cands: []AlbumCandidate{
			{Title: "To Pimp A Butterfly", PrimaryArtist: "Kendrick Lamar", Year: &y, Provider: "spotify", ProviderRef: "a"},
			{Title: "To Pimp A Butterfly", PrimaryArtist: "Kendrick Lamar", Year: &y, Provider: "itunes", ProviderRef: "b"},
		}},
	}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "butterfly")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeSaved {
		t.Fatalf("want OutcomeSaved, got %v text %q", um.Outcome, um.Text)
	}
	if st.insertCalls != 1 {
		t.Fatalf("inserts %d (want 1, no disambig session)", st.insertCalls)
	}
	if len(st.disambigRaw) != 0 {
		t.Fatalf("unexpected disambig session stored")
	}
}

func TestProcessAlbumQuery_DisambigStoresTwo(t *testing.T) {
	t.Parallel()
	y2012, y1971 := 2012, 1971
	st := &memStore{}
	svc := &SaveService{
		Store: st,
		Search: &fakeSearch{cands: []AlbumCandidate{
			{Title: "Red", PrimaryArtist: "Taylor Swift", Year: &y2012, Provider: "a", ProviderRef: "1"},
			{Title: "Red", PrimaryArtist: "Gil Scott-Heron", Year: &y1971, Provider: "a", ProviderRef: "2"},
			{Title: "Extra", PrimaryArtist: "X", Provider: "a", ProviderRef: "3"},
		}},
	}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "red")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeDisambig {
		t.Fatalf("outcome %v", um.Outcome)
	}
	if um.PickCount != 2 {
		t.Fatalf("pickCount %d", um.PickCount)
	}
	if len(um.AlbumButtonLabels) != 2 {
		t.Fatalf("labels %v", um.AlbumButtonLabels)
	}
	var got []AlbumCandidate
	if err := json.Unmarshal(st.disambigRaw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("stored %d candidates", len(got))
	}
	if st.insertCalls != 0 {
		t.Fatalf("inserts before pick: %d", st.insertCalls)
	}
}

func TestProcessPickByIndex_SecondAlbum(t *testing.T) {
	t.Parallel()
	y1, y2 := 2012, 1971
	raw, _ := json.Marshal([]AlbumCandidate{
		{Title: "Red", PrimaryArtist: "Taylor Swift", Year: &y1, Provider: "a", ProviderRef: "1"},
		{Title: "Red", PrimaryArtist: "Gil Scott-Heron", Year: &y2, Provider: "a", ProviderRef: "2"},
	})
	st := &memStore{
		sessionID:   "sess",
		disambigRaw: raw,
		listenerID:  "11111111-1111-1111-1111-111111111111",
	}
	svc := &SaveService{Store: st, Search: &fakeSearch{}}
	um, err := svc.ProcessPickByIndex(context.Background(), "telegram", "1", "a", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeSaved {
		t.Fatalf("outcome %v", um.Outcome)
	}
	if st.insertCalls != 1 || st.deleteSessCalls != 1 {
		t.Fatalf("inserts %d deleteSess %d", st.insertCalls, st.deleteSessCalls)
	}
}

func TestProcessPickByIndex_OtherNoInsert(t *testing.T) {
	t.Parallel()
	y1, y2 := 2012, 1971
	raw, _ := json.Marshal([]AlbumCandidate{
		{Title: "Red", PrimaryArtist: "Taylor Swift", Year: &y1, Provider: "a", ProviderRef: "1"},
		{Title: "Red", PrimaryArtist: "Gil Scott-Heron", Year: &y2, Provider: "a", ProviderRef: "2"},
	})
	st := &memStore{
		sessionID:   "sess",
		disambigRaw: raw,
		listenerID:  "11111111-1111-1111-1111-111111111111",
	}
	svc := &SaveService{Store: st, Search: &fakeSearch{}}
	um, err := svc.ProcessPickByIndex(context.Background(), "telegram", "1", "a", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeRefineQuery {
		t.Fatalf("outcome %v text %q", um.Outcome, um.Text)
	}
	if st.insertCalls != 0 {
		t.Fatalf("inserts %d", st.insertCalls)
	}
	if st.deleteSessCalls != 1 {
		t.Fatalf("expected session cleared, deleteSess %d", st.deleteSessCalls)
	}
	if um.Text == "" {
		t.Fatal("empty refine text")
	}
}

func TestProcessAlbumQuery_NoMatch(t *testing.T) {
	t.Parallel()
	svc := &SaveService{Store: &memStore{}, Search: &fakeSearch{err: ErrNoMatch}}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeNoMatch {
		t.Fatalf("outcome %v", um.Outcome)
	}
}

func TestProcessAlbumQuery_ProviderExhausted(t *testing.T) {
	t.Parallel()
	svc := &SaveService{Store: &memStore{}, Search: &fakeSearch{err: ErrAllProvidersExhausted}}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeProviderExhausted {
		t.Fatalf("outcome %v", um.Outcome)
	}
}

func TestProcessAlbumQuery_TransientError(t *testing.T) {
	t.Parallel()
	svc := &SaveService{Store: &memStore{}, Search: &fakeSearch{err: errors.New("network")}}
	um, err := svc.ProcessAlbumQuery(context.Background(), "telegram", "1", "a", "", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if um.Outcome != OutcomeTransientError {
		t.Fatalf("outcome %v", um.Outcome)
	}
}

func TestFormatAlbumLine(t *testing.T) {
	t.Parallel()
	y := 2000
	got := formatAlbumLine(AlbumCandidate{Title: "OK Computer", PrimaryArtist: "Radiohead", Year: &y})
	if got != "OK Computer | Radiohead (2000)" {
		t.Fatalf("got %q", got)
	}
}
