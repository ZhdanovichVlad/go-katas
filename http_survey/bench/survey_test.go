package bench_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/ZhdanovichVlad/go-katas/http_survey/semaphore"
	"github.com/ZhdanovichVlad/go-katas/http_survey/worker_pool"
)

// surveyFn — общий контракт обеих реализаций.
type surveyFn func(ctx context.Context, urls []string, parallel int, isMock bool) []int

// Слайс, а не map — чтобы порядок прогона бенчмарков был стабильным
// (по map Go итерируется в случайном порядке).
var implementations = []struct {
	name string
	fn   surveyFn
}{
	{"semaphore", semaphore.Survey},
	{"worker_pool", worker_pool.Survey},
}

// makeURLs возвращает n урлов, среди которых ровно `unique` различных.
// Это позволяет управлять процентом попаданий в кеш.
func makeURLs(n, unique int) []string {
	if unique <= 0 {
		unique = 1
	}
	urls := make([]string, n)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.test/%d", i%unique)
	}
	return urls
}

// TestSurveyCorrectness — табличный тест: длина результата и значения статусов
// для каждой реализации на одних и тех же входах.
func TestSurveyCorrectness(t *testing.T) {
	cases := []struct {
		name     string
		urls     []string
		parallel int
		want     int
	}{
		{"empty", nil, 4, 0},
		{"one", []string{"https://a"}, 1, 1},
		{"unique_5", makeURLs(5, 5), 2, 5},
		{"dup_10_in_3", makeURLs(10, 3), 4, 10},
		{"parallel_gt_urls", makeURLs(3, 3), 16, 3},
		{"big", makeURLs(100, 70), 8, 100},
	}

	for _, impl := range implementations {
		for _, c := range cases {
			t.Run(impl.name+"/"+c.name, func(t *testing.T) {
				got := impl.fn(context.Background(), c.urls, c.parallel, true)
				if len(got) != c.want {
					t.Fatalf("len(got) = %d, want %d", len(got), c.want)
				}
				for i, code := range got {
					if code != 200 {
						t.Errorf("got[%d] = %d, want 200", i, code)
					}
				}
			})
		}
	}
}

// TestSurveyEquivalence проверяет, что обе реализации возвращают
// один и тот же срез статусов на одинаковом входе.
// Порядок гарантируется контрактом Survey ("вернет статусы в том же порядке,
// в котором они были переданы"), поэтому сравнение строгое — slices.Equal.
func TestSurveyEquivalence(t *testing.T) {
	urls := makeURLs(50, 10)
	ctx := context.Background()

	gotSem := semaphore.Survey(ctx, urls, 8, true)
	gotWP := worker_pool.Survey(ctx, urls, 8, true)

	if !slices.Equal(gotSem, gotWP) {
		t.Errorf("результаты различаются:\n sem=%v\n wp =%v", gotSem, gotWP)
	}
}

// BenchmarkSurvey гоняет обе реализации по одной матрице конфигов:
//   - размер пачки урлов,
//   - доля уникальных (управление процентом кеш-хитов),
//   - уровень параллелизма.
//
// ВАЖНО: мок http_client спит 1ms на каждый Get — поэтому абсолютные числа
// в первую очередь отражают latency мока, а не «чистый» оверхед примитивов
// синхронизации. Сравнивать имеет смысл relative-разницу между semaphore
// и worker_pool в одной и той же ячейке матрицы.
func BenchmarkSurvey(b *testing.B) {
	type cfg struct {
		urls     int
		unique   int
		parallel int
	}
	cfgs := []cfg{
		{urls: 100, unique: 100, parallel: 1},
		{urls: 100, unique: 100, parallel: 8},
		{urls: 100, unique: 100, parallel: 32},
		{urls: 100, unique: 100, parallel: 128},
		{urls: 1000, unique: 1000, parallel: 8},
		{urls: 1000, unique: 1000, parallel: 64},
		{urls: 1000, unique: 1000, parallel: 256},
		{urls: 1000, unique: 50, parallel: 64}, // высокая доля кеш-хитов
	}

	for _, impl := range implementations {
		for _, c := range cfgs {
			name := fmt.Sprintf("%s/urls=%d/uniq=%d/par=%d",
				impl.name, c.urls, c.unique, c.parallel)
			urls := makeURLs(c.urls, c.unique)
			b.Run(name, func(b *testing.B) {
				ctx := context.Background()
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = impl.fn(ctx, urls, c.parallel, true)
				}
			})
		}
	}
}
