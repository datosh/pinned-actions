document.addEventListener('DOMContentLoaded', () => {
    const YEARS = ['2024', '2025', '2026'];
    let chart = null;

    function updateChart(year) {
        fetch('pinned.json')
            .then(response => response.json())
            .then(data => {
                const yearData = data[year];
                const total = yearData.pinned + yearData['partially-pinned'] + yearData.unpinned;
                const ctx = document.getElementById('pinnedActionsChart').getContext('2d');

                if (chart) {
                    chart.destroy();
                }

                chart = new Chart(ctx, {
                    type: 'doughnut',
                    data: {
                        labels: ['Fully Pinned', 'Partially Pinned', 'Unpinned'],
                        datasets: [{
                            data: [yearData.pinned, yearData['partially-pinned'], yearData.unpinned],
                            backgroundColor: ['#5D8234', '#FFD23F', '#C64756'],
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
            })
            .catch(error => console.error('Error fetching the JSON data:', error));
    }

    const slider = document.getElementById('yearSlider');
    const selectedYearLabel = document.getElementById('selectedYear');

    // Initialize with 2026 (index 2)
    updateChart(YEARS[slider.value]);

    slider.addEventListener('input', function() {
        const year = YEARS[this.value];
        selectedYearLabel.textContent = year;
        updateChart(year);
    });
});
