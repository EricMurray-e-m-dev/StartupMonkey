// BarberBook - My Bookings Page
// AI-generated booking search logic

const API_BASE_URL = '/api';

document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('search-form');
    form.addEventListener('submit', handleSearch);
});

// Handle search form submission
async function handleSearch(event) {
    event.preventDefault();
    
    const phone = document.getElementById('search_phone').value;
    const searchText = document.getElementById('search-text');
    const searchSpinner = document.getElementById('search-spinner');
    const resultsContainer = document.getElementById('results-container');
    const noResults = document.getElementById('no-results');
    const bookingsList = document.getElementById('bookings-list');
    
    // Show loading state
    searchText.classList.add('d-none');
    searchSpinner.classList.remove('d-none');
    resultsContainer.classList.add('d-none');
    noResults.classList.add('d-none');
    
    try {
        // INTENTIONAL FLAW: This endpoint does full table scan on customer_phone
        const response = await fetch(`${API_BASE_URL}/bookings/search?phone=${encodeURIComponent(phone)}`);
        
        if (!response.ok) {
            throw new Error('Search failed');
        }
        
        const bookings = await response.json();
        
        if (bookings.length === 0) {
            noResults.classList.remove('d-none');
        } else {
            // Render bookings
            bookingsList.innerHTML = bookings.map(booking => `
                <div class="card mb-3">
                    <div class="card-body">
                        <div class="row">
                            <div class="col-md-8">
                                <h5 class="card-title">${booking.service}</h5>
                                <p class="card-text">
                                    <strong>Date:</strong> ${new Date(booking.booking_date).toLocaleDateString()}<br>
                                    <strong>Time:</strong> ${booking.booking_time}<br>
                                    <strong>Booking ID:</strong> ${booking.booking_id}
                                </p>
                            </div>
                            <div class="col-md-4 text-md-end">
                                <span class="badge ${getStatusBadgeClass(booking.status)} mb-2">
                                    ${booking.status.toUpperCase()}
                                </span>
                                <p class="mb-0"><strong>€${parseFloat(booking.price).toFixed(2)}</strong></p>
                            </div>
                        </div>
                    </div>
                </div>
            `).join('');
            
            resultsContainer.classList.remove('d-none');
        }
        
    } catch (error) {
        console.error('Error searching bookings:', error);
        alert('Failed to search bookings. Please try again.');
    } finally {
        // Reset button state
        searchText.classList.remove('d-none');
        searchSpinner.classList.add('d-none');
    }
}

// Get badge class based on status
function getStatusBadgeClass(status) {
    switch (status.toLowerCase()) {
        case 'confirmed':
            return 'bg-primary';
        case 'completed':
            return 'bg-success';
        case 'cancelled':
            return 'bg-danger';
        default:
            return 'bg-secondary';
    }
}