document.addEventListener('DOMContentLoaded', () => {
    const slider = document.getElementById('yearSlider');
    const selectedYearLabel = document.getElementById('selectedYear');
    const yearLabels = document.getElementById('yearLabels');
    let chart = null;

    // The slices of a year, greens first. "Fully Locked" only exists for years
    // scanned with lockfile data (recognizable by with-lockfile); older years
    // keep their three slices.
    function segments(yearData) {
        const slices = [{ label: 'Fully Pinned', count: yearData.pinned, color: '#5D8234' }];
        if (yearData['with-lockfile'] !== undefined) {
            slices.push({ label: 'Fully Locked', count: yearData.locked || 0, color: '#8FBF5A' });
        }
        slices.push(
            { label: 'Partially Pinned', count: yearData['partially-pinned'], color: '#FFD23F' },
            { label: 'Unpinned', count: yearData.unpinned, color: '#C64756' },
        );
        return slices;
    }

    function totalUsingActions(yearData) {
        return segments(yearData).reduce((sum, slice) => sum + slice.count, 0);
    }

    function renderChart(yearData) {
        const slices = segments(yearData);
        const total = totalUsingActions(yearData);
        const ctx = document.getElementById('pinnedActionsChart').getContext('2d');

        if (chart) {
            chart.destroy();
        }

        chart = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: slices.map(slice => slice.label),
                datasets: [{
                    data: slices.map(slice => slice.count),
                    backgroundColor: slices.map(slice => slice.color),
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: { position: 'bottom' },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const count = context.raw;
                                const percentage = ((count / total) * 100).toFixed(2);
                                return `${context.label}: ${count} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }

    // Fills the numbers in the text from the newest year. The HTML keeps the
    // last hand-written values, which stay when the data lacks a field.
    function updateSummary(latest) {
        const using = totalUsingActions(latest);
        document.getElementById('fullyPinnedPercent').textContent = Math.round(latest.pinned / using * 100);

        if (latest.scanned) {
            document.getElementById('lastUpdated').textContent = new Date(latest.scanned + 'T00:00:00Z')
                .toLocaleString('en-US', { month: 'long', year: 'numeric', timeZone: 'UTC' });
        }
        if (latest['no-actions']) {
            const all = using + latest['no-actions'];
            document.getElementById('noActionsPercent').textContent = Math.round(latest['no-actions'] / all * 100);
        }
    }

    // The years on the slider are whatever pinned.json contains, oldest first.
    fetch('pinned.json')
        .then(response => response.json())
        .then(data => {
            const years = Object.keys(data).sort();
            slider.max = years.length - 1;
            slider.value = years.length - 1;
            yearLabels.replaceChildren(...years.map(year => {
                const label = document.createElement('span');
                label.textContent = year;
                return label;
            }));

            const show = year => {
                selectedYearLabel.textContent = year;
                renderChart(data[year]);
            };
            updateSummary(data[years[years.length - 1]]);
            show(years[slider.value]);
            slider.addEventListener('input', () => show(years[slider.value]));
        })
        .catch(error => console.error('Error fetching the JSON data:', error));
});
