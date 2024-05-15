document.addEventListener('DOMContentLoaded', () => {
    fetch('pinned.json')
        .then(response => response.json())
        .then(data => {
            const total = data.pinned + data['partially-pinned'] + data.unpinned;
            const ctx = document.getElementById('pinnedActionsChart').getContext('2d');
            const chartData = {
                labels: ['Pinned', 'Partially-pinned', 'Un-pinned'],
                datasets: [{
                    data: [data.pinned, data['partially-pinned'], data.unpinned],
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
            new Chart(ctx, config);
        })
        .catch(error => console.error('Error fetching the JSON data:', error));
});
