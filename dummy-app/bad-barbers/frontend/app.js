// BarberBook - Homepage JavaScript
// AI-generated code for loading popular barbers and services

const API_BASE_URL = 'http://localhost:3001/api';

// Load popular barbers on page load
document.addEventListener('DOMContentLoaded', () => {
    loadPopularBarbers();
    loadServices();
});

// Load popular barbers (triggers cache miss detection)
async function loadPopularBarbers() {
    const container = document.getElementById('popular-barbers-container');
    
    try {
        const response = await fetch(`${API_BASE_URL}/popular-barbers`);
        
        if (!response.ok) {
            throw new Error('Failed to load popular barbers');
        }
        
        const barbers = await response.json();
        
        if (barbers.length === 0) {
            container.innerHTML = '<p class="text-center">No barbers available.</p>';
            return;
        }
        
        // Render barber cards
        container.innerHTML = barbers.map(barber => `
            <div class="col-md-6 col-lg-3 mb-4">
                <div class="card h-100 shadow-sm">
                    <div class="card-body text-center">
                        <div class="mb-3">
                            <div class="rounded-circle bg-primary text-white d-inline-flex align-items-center justify-content-center" 
                                 style="width: 80px; height: 80px; font-size: 2rem;">
                                ${barber.barber_name.charAt(0)}
                            </div>
                        </div>
                        <h5 class="card-title">${barber.barber_name}</h5>
                        <p class="card-text text-muted">${barber.speciality}</p>
                        <div class="mb-2">
                            <span class="badge bg-warning text-dark">
                                ⭐ ${barber.rating}
                            </span>
                        </div>
                        <p class="text-muted small">${barber.total_bookings} completed bookings</p>
                        <a href="booking.html" class="btn btn-primary btn-sm">Book Now</a>
                    </div>
                </div>
            </div>
        `).join('');
        
    } catch (error) {
        console.error('Error loading popular barbers:', error);
        container.innerHTML = `
            <div class="col-12">
                <div class="alert alert-danger" role="alert">
                    Failed to load barbers. Please try again later.
                </div>
            </div>
        `;
    }
}

// Load services
async function loadServices() {
    const container = document.getElementById('services-container');
    
    try {
        const response = await fetch(`${API_BASE_URL}/services`);
        
        if (!response.ok) {
            throw new Error('Failed to load services');
        }
        
        const services = await response.json();
        
        if (services.length === 0) {
            container.innerHTML = '<p class="text-center">No services available.</p>';
            return;
        }
        
        // Render service cards
        container.innerHTML = services.map(service => `
            <div class="col-md-6 col-lg-4 mb-4">
                <div class="card h-100">
                    <div class="card-body">
                        <h5 class="card-title">${service.service_name}</h5>
                        <p class="card-text text-muted">${service.description}</p>
                        <div class="d-flex justify-content-between align-items-center mt-3">
                            <span class="h5 mb-0 text-primary">€${parseFloat(service.price).toFixed(2)}</span>
                            <span class="text-muted small">${service.duration_minutes} mins</span>
                        </div>
                    </div>
                </div>
            </div>
        `).join('');
        
    } catch (error) {
        console.error('Error loading services:', error);
        container.innerHTML = `
            <div class="col-12">
                <div class="alert alert-danger" role="alert">
                    Failed to load services. Please try again later.
                </div>
            </div>
        `;
    }
}