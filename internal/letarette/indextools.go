package letarette

import (
	"context"

	"github.com/erkkah/letarette/internal/snowball"
)

type Stats struct {
	Spaces []struct {
		Name  string
		State InterestListState
	}
	CommonTerms []struct {
		Term  string
		Count int
	}
	Terms   int
	Docs    int
	Stemmer snowball.Settings
}

func GetIndexStats(db Database) (Stats, error) {
	var s Stats

	sql := db.getRawDB()

	var err error

	ctx := context.Background()
	conn, err := sql.Conn(ctx)
	if err != nil {
		return s, err
	}

	s.Stemmer, _, _ = db.getStemmerState()

	rows, err := conn.QueryContext(ctx, `select space from spaces`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var space string
		rows.Scan(&space)

		state, err := db.getInterestListState(ctx, space)
		if err != nil {
			return s, err
		}
		s.Spaces = append(s.Spaces, struct {
			Name  string
			State InterestListState
		}{space, state})
	}

	_, err = conn.ExecContext(
		ctx,
		`create virtual table temp.stats using fts5vocab(main, 'fts', 'row');`,
	)
	if err != nil {
		return s, err
	}

	rows, err = conn.QueryContext(
		ctx,
		`select term, sum(cnt) as num from temp.stats group by term order by num desc limit 15;`,
	)
	if err != nil {
		return s, err
	}

	for rows.Next() {
		var term string
		var count int
		err = rows.Scan(&term, &count)
		if err != nil {
			return s, err
		}
		s.CommonTerms = append(s.CommonTerms, struct {
			Term  string
			Count int
		}{term, count})
	}

	row := conn.QueryRowContext(
		ctx,
		`select count(distinct term) from temp.stats`,
	)
	row.Scan(&s.Terms)

	row = conn.QueryRowContext(
		ctx,
		`select count(*) from docs`,
	)
	row.Scan(&s.Docs)

	return s, nil
}

func CheckIndex(db Database) error {
	sql := db.getRawDB()
	_, err := sql.Exec(`insert into fts(fts) values("integrity-check");`)
	if err != nil {
		return err
	}
	return nil
}

func RebuildIndex(db Database) error {
	sql := db.getRawDB()
	_, err := sql.Exec(`insert into fts(fts) values("rebuild");`)
	if err != nil {
		return err
	}
	return nil
}

func ForceIndexStemmerState(state snowball.Settings, db Database) error {
	return db.setStemmerState(state)
}
