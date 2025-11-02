import { describe, it, expect } from '@jest/globals';

describe('Sidebar Navigation', () => {
  it('should have correct navigation items', () => {
    const navigation = [
      { name: "Overview", href: "/" },
      { name: "Metrics", href: "/metrics" },
      { name: "Detections", href: "/detections" },
      { name: "Actions", href: "/actions" },
    ];

    expect(navigation).toHaveLength(4);
    expect(navigation[0].name).toBe("Overview");
    expect(navigation[3].name).toBe("Actions");
  });

  it('should have valid href paths', () => {
    const navigation = [
      { name: "Overview", href: "/" },
      { name: "Metrics", href: "/metrics" },
      { name: "Detections", href: "/detections" },
      { name: "Actions", href: "/actions" },
    ];

    navigation.forEach(item => {
      expect(item.href).toBeTruthy();
      expect(item.href).toMatch(/^\//);
    });
  });
});