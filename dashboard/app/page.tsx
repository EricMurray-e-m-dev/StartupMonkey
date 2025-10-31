import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function Home() {
  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <Card className="w-96">
        <CardHeader>
          <CardTitle>StartupMonkey Dashboard</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-muted-foreground">
            Foundation Setup
          </p>

          <div className="flex gap-2">
            <Badge>Collector</Badge>
            <Badge variant={"secondary"}>Analyser</Badge>
            <Badge variant={"outline"}>Executor</Badge>
          </div>

          <Button className="w-full">
            Begin
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
