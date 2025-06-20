document.addEventListener('DOMContentLoaded', () => {
    let chart = null;

    function updateChart(year) {
        fetch('pinned.json')
            .then(response => response.json())
            .then(data => {
                const yearData = data[year];
                const total = yearData.pinned + yearData['partially-pinned'] + yearData.unpinned;
                const ctx = document.getElementById('pinnedActionsChart').getContext('2d');

                // Destroy existing chart if it exists
                if (chart) {
                    chart.destroy();
                }

                const chartData = {
                    labels: ['Fully Pinned', 'Partially Pinned', 'Unpinned'],
                    datasets: [{
                        data: [yearData.pinned, yearData['partially-pinned'], yearData.unpinned],
                        backgroundColor: ['#5D8234', '#FFD23F', '#C64756'],
                        // borderColor: ['#10B981', '#FBBF24', '#F87171'],
                        borderWidth: 1
                    }]
                };
                const config = {
                    type: 'doughnut',
                    data: chartData,
                    options: {
                        responsive: true,
                        plugins: {
                            legend: {
                                position: 'bottom',
                            },
                            tooltip: {
                                callbacks: {
                                    label: function(context) {
                                        let label = context.label || '';
                                        if (label) {
                                            label += ': ';
                                        }
                                        const count = context.raw;
                                        const percentage = ((count / total) * 100).toFixed(2);
                                        label += `${count} (${percentage}%)`;
                                        return label;
                                    }
                                }
                            }
                        }
                    }
                };
                chart = new Chart(ctx, config);
            })
            .catch(error => console.error('Error fetching the JSON data:', error));
    }

    // Initialize with 2025 data (default)
    updateChart('2025');

    // Set toggle to checked (2025) by default
    document.getElementById('yearToggle').checked = true;

    // Add event listener for year toggle
    document.getElementById('yearToggle').addEventListener('change', function() {
        const year = this.checked ? '2025' : '2024';
        updateChart(year);
    });
});
